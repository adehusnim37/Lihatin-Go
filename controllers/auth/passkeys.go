package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	pkgauth "github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/middleware"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type passkeyUser struct {
	account     *user.User
	credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte {
	sum := sha256.Sum256([]byte("lihatin-passkey:" + u.account.ID))
	return sum[:]
}
func (u *passkeyUser) WebAuthnName() string                       { return u.account.Email }
func (u *passkeyUser) WebAuthnDisplayName() string                { return u.account.Username }
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

type passkeyChallenge struct {
	Kind      string               `json:"kind"`
	UserID    string               `json:"user_id"`
	SessionID string               `json:"session_id"`
	Name      string               `json:"name"`
	Data      webauthn.SessionData `json:"data"`
}

func passkeyEngine() (*webauthn.WebAuthn, error) {
	frontend, err := url.Parse(config.GetEnvOrDefault(config.EnvFrontendURL, "http://localhost:3000"))
	if err != nil || frontend.Hostname() == "" {
		return nil, errors.New("invalid frontend origin")
	}
	rpID := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_ID"))
	if rpID == "" {
		rpID = frontend.Hostname()
	}
	origins := []string{frontend.Scheme + "://" + frontend.Host}
	if raw := strings.TrimSpace(os.Getenv("WEBAUTHN_ORIGINS")); raw != "" {
		origins = nil
		for _, value := range strings.Split(raw, ",") {
			parsed, e := url.Parse(strings.TrimSpace(value))
			if e != nil || parsed.Hostname() == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
				return nil, errors.New("invalid WebAuthn origin")
			}
			origins = append(origins, parsed.Scheme+"://"+parsed.Host)
		}
	}
	return webauthn.New(&webauthn.Config{RPID: rpID, RPDisplayName: "Lihat.in", RPOrigins: origins, AuthenticatorSelection: protocol.AuthenticatorSelection{ResidentKey: protocol.ResidentKeyRequirementRequired, UserVerification: protocol.VerificationRequired}})
}
func securityRedis() (*redis.Client, error) {
	manager := middleware.GetSessionManager()
	if manager == nil || manager.GetRedisClient() == nil {
		return nil, errors.New("security service unavailable")
	}
	return manager.GetRedisClient(), nil
}
func randomSecurityToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}
func securityError(ctx *gin.Context, status int, message string) {
	httputil.SendErrorResponse(ctx, status, "SECURITY_VERIFICATION_FAILED", message, "auth")
}
func (c *Controller) loadPasskeyUser(id string) (*passkeyUser, error) {
	account, err := c.repo.GetUserRepository().GetUserByID(id)
	if err != nil {
		return nil, err
	}
	var rows []user.Passkey
	if err = c.GormDB.Where("user_id = ?", id).Find(&rows).Error; err != nil {
		return nil, err
	}
	u := &passkeyUser{account: account}
	for _, row := range rows {
		var cred webauthn.Credential
		if err = json.Unmarshal([]byte(row.CredentialData), &cred); err != nil {
			return nil, err
		}
		u.credentials = append(u.credentials, cred)
	}
	return u, nil
}
func (c *Controller) saveChallenge(ctx *gin.Context, value passkeyChallenge, options any) {
	client, err := securityRedis()
	if err != nil {
		securityError(ctx, 503, "Security service unavailable. Try again later.")
		return
	}
	token, err := randomSecurityToken()
	if err != nil {
		ctx.AbortWithStatus(500)
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		ctx.AbortWithStatus(500)
		return
	}
	if err = client.Set(ctx.Request.Context(), "webauthn:"+token, data, 5*time.Minute).Err(); err != nil {
		securityError(ctx, 503, "Security service unavailable")
		return
	}
	ctx.Header("Cache-Control", "no-store")
	httputil.SendOKResponse(ctx, gin.H{"challenge_id": token, "options": options}, "Continue with your passkey")
}
func (c *Controller) finishChallenge(ctx *gin.Context, kind string) (*passkeyChallenge, *http.Request, error) {
	var body struct {
		ChallengeID string          `json:"challenge_id" binding:"required,len=64,hexadecimal"`
		Credential  json.RawMessage `json:"credential" binding:"required"`
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 1<<20)
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return nil, nil, err
	}
	client, err := securityRedis()
	if err != nil {
		return nil, nil, err
	}
	data, err := client.GetDel(ctx.Request.Context(), "webauthn:"+body.ChallengeID).Bytes()
	if err != nil {
		return nil, nil, err
	}
	var value passkeyChallenge
	if err = json.Unmarshal(data, &value); err != nil {
		return nil, nil, err
	}
	if value.Kind != kind {
		return nil, nil, errors.New("wrong challenge purpose")
	}
	if kind != "login" && (value.UserID != ctx.GetString("user_id") || value.SessionID != ctx.GetString("session_id")) {
		return nil, nil, errors.New("challenge does not belong to this session")
	}
	request := ctx.Request.Clone(ctx.Request.Context())
	request.Body = io.NopCloser(bytes.NewReader(body.Credential))
	return &value, request, nil
}
func (c *Controller) issueSecurityGrant(ctx *gin.Context, mfa bool) {
	client, err := securityRedis()
	if err != nil {
		securityError(ctx, 503, "Security service unavailable")
		return
	}
	token, err := randomSecurityToken()
	if err != nil {
		ctx.AbortWithStatus(500)
		return
	}
	value := ctx.GetString("user_id") + ":" + ctx.GetString("session_id")
	if mfa {
		value += ":mfa"
	}
	if err = client.Set(ctx.Request.Context(), "security_grant:"+token, value, 5*time.Minute).Err(); err != nil {
		securityError(ctx, 503, "Security service unavailable")
		return
	}
	ctx.Header("Cache-Control", "no-store")
	httputil.SendOKResponse(ctx, gin.H{"verification_token": token}, "Identity verified")
}
func (c *Controller) consumeSecurityGrant(ctx *gin.Context, token string) bool {
	client, err := securityRedis()
	if err != nil {
		return false
	}
	return pkgauth.ConsumeSecurityGrant(ctx.Request.Context(), client, ctx.GetString("user_id"), ctx.GetString("session_id"), token)
}

// VerifySecurityAction requires the existing second factor when enabled, otherwise the account password.
func (c *Controller) VerifySecurityAction(ctx *gin.Context) {
	var req dto.DisableTOTPRequest
	if ctx.ShouldBindJSON(&req) != nil {
		securityError(ctx, 400, "Enter a verification code or password")
		return
	}
	ua, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(ctx.GetString("user_id"))
	if err != nil {
		ctx.AbortWithStatus(401)
		return
	}
	valid := false
	if ua.HasEnabledTOTP() {
		encrypted, e := c.repo.GetAuthMethodRepository().GetTOTPSecret(ua.ID)
		if e == nil {
			secret, e := pkgauth.DecryptTOTPSecret(encrypted)
			valid = e == nil && pkgauth.ValidateTOTPCodeWithWindow(secret, req.TOTPCode, 1)
		}
	} else {
		valid = req.Password != "" && pkgauth.CheckPassword(ua.PasswordHash, req.Password) == nil
	}
	if !valid {
		securityError(ctx, 401, "Verification failed. Use your authenticator code, or your password if 2FA is off.")
		return
	}
	c.issueSecurityGrant(ctx, ua.HasEnabledTOTP())
}
func (c *Controller) ListPasskeys(ctx *gin.Context) {
	var rows []user.Passkey
	if err := c.GormDB.Where("user_id = ?", ctx.GetString("user_id")).Order("created_at desc").Find(&rows).Error; err != nil {
		httputil.HandleError(ctx, err, "passkeys")
		return
	}
	_, err := passkeyEngine()
	httputil.SendOKResponse(ctx, gin.H{"passkeys": rows, "available": err == nil}, "Passkeys retrieved")
}
func (c *Controller) BeginPasskeyRegistration(ctx *gin.Context) {
	var body struct {
		Name  string `json:"name" binding:"required,max=100"`
		Token string `json:"verification_token" binding:"required"`
	}
	if ctx.ShouldBindJSON(&body) != nil || strings.TrimSpace(body.Name) == "" {
		securityError(ctx, 400, "Give this passkey a name")
		return
	}
	if !c.consumeSecurityGrant(ctx, body.Token) {
		securityError(ctx, 401, "Verify your identity again before adding a passkey")
		return
	}
	engine, err := passkeyEngine()
	if err != nil {
		securityError(ctx, 503, "Passkeys are not configured")
		return
	}
	u, err := c.loadPasskeyUser(ctx.GetString("user_id"))
	if err != nil {
		ctx.AbortWithStatus(500)
		return
	}
	if len(u.credentials) >= 10 {
		securityError(ctx, 400, "You can register up to 10 passkeys")
		return
	}
	options, data, err := engine.BeginRegistration(u, webauthn.WithExclusions(webauthn.Credentials(u.credentials).CredentialDescriptors()))
	if err != nil {
		ctx.AbortWithStatus(500)
		return
	}
	c.saveChallenge(ctx, passkeyChallenge{Kind: "register", UserID: u.account.ID, SessionID: ctx.GetString("session_id"), Name: strings.TrimSpace(body.Name), Data: *data}, options)
}
func (c *Controller) FinishPasskeyRegistration(ctx *gin.Context) {
	challenge, request, err := c.finishChallenge(ctx, "register")
	if err != nil {
		securityError(ctx, 400, "Passkey setup expired. Start again.")
		return
	}
	engine, err := passkeyEngine()
	if err != nil {
		ctx.AbortWithStatus(503)
		return
	}
	u, err := c.loadPasskeyUser(challenge.UserID)
	if err != nil {
		ctx.AbortWithStatus(500)
		return
	}
	credential, err := engine.FinishRegistration(u, challenge.Data, request)
	if err != nil {
		securityError(ctx, 400, "Could not verify this passkey. Start again.")
		return
	}
	encoded, err := json.Marshal(credential)
	if err != nil {
		ctx.AbortWithStatus(500)
		return
	}
	sum := sha256.Sum256(credential.ID)
	row := user.Passkey{ID: uuid.NewString(), UserID: u.account.ID, Name: challenge.Name, CredentialID: base64.RawURLEncoding.EncodeToString(credential.ID), CredentialHash: hex.EncodeToString(sum[:]), CredentialData: string(encoded)}
	err = c.GormDB.Transaction(func(tx *gorm.DB) error {
		var owner user.UserAuth
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", u.account.ID).First(&owner).Error; e != nil {
			return e
		}
		var count int64
		if e := tx.Model(&user.Passkey{}).Where("user_id = ?", u.account.ID).Count(&count).Error; e != nil {
			return e
		}
		if count >= 10 {
			return errors.New("passkey limit reached")
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		securityError(ctx, 400, "Could not save passkey. It may already be registered.")
		return
	}
	c.passkeyNotice(u.account, "added", row.Name)
	httputil.SendCreatedResponse(ctx, row, "Passkey added")
}
func (c *Controller) RenamePasskey(ctx *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required,max=100"`
	}
	if ctx.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.Name) == "" {
		securityError(ctx, 400, "Enter a name up to 100 characters")
		return
	}
	result := c.GormDB.Model(&user.Passkey{}).Where("id = ? AND user_id = ?", ctx.Param("id"), ctx.GetString("user_id")).Update("name", strings.TrimSpace(req.Name))
	if result.Error != nil {
		ctx.AbortWithStatus(500)
		return
	}
	if result.RowsAffected == 0 {
		ctx.AbortWithStatus(404)
		return
	}
	httputil.SendOKResponse(ctx, nil, "Passkey renamed")
}
func (c *Controller) DeletePasskey(ctx *gin.Context) {
	var req struct {
		Token string `json:"verification_token" binding:"required"`
	}
	if ctx.ShouldBindJSON(&req) != nil || !c.consumeSecurityGrant(ctx, req.Token) {
		securityError(ctx, 401, "Verify your identity again before removing a passkey")
		return
	}
	var row user.Passkey
	if c.GormDB.Where("id = ? AND user_id = ?", ctx.Param("id"), ctx.GetString("user_id")).First(&row).Error != nil {
		ctx.AbortWithStatus(404)
		return
	}
	err := c.GormDB.Transaction(func(tx *gorm.DB) error {
		var owner user.UserAuth
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", row.UserID).First(&owner).Error; e != nil {
			return e
		}
		var count int64
		if e := tx.Model(&user.Passkey{}).Where("user_id = ?", row.UserID).Count(&count).Error; e != nil {
			return e
		}
		if count <= 1 && owner.PasswordHash == "" {
			var google int64
			if e := tx.Model(&user.AuthMethod{}).Where("user_auth_id = ? AND type = ? AND is_enabled = ? AND is_verified = ? AND deleted_at IS NULL", owner.ID, user.AuthMethodTypeOAuthGoogle, true, true).Count(&google).Error; e != nil {
				return e
			}
			if google == 0 {
				return errors.New("keep at least one sign-in method before removing your last passkey")
			}
		}
		return tx.Delete(&row).Error
	})
	if err != nil {
		securityError(ctx, 400, "Keep another sign-in method before removing your last passkey")
		return
	}
	if account, err := c.repo.GetUserRepository().GetUserByID(row.UserID); err == nil {
		c.passkeyNotice(account, "removed", row.Name)
	}
	httputil.SendOKResponse(ctx, nil, "Passkey removed")
}
func (c *Controller) BeginPasskeyLogin(ctx *gin.Context) {
	engine, err := passkeyEngine()
	if err != nil {
		securityError(ctx, 503, "Passkeys are not configured. Use another sign-in method.")
		return
	}
	options, data, err := engine.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		ctx.AbortWithStatus(500)
		return
	}
	c.saveChallenge(ctx, passkeyChallenge{Kind: "login", Data: *data}, options)
}
func (c *Controller) BeginPasskeyReauth(ctx *gin.Context) {
	engine, err := passkeyEngine()
	if err != nil {
		ctx.AbortWithStatus(503)
		return
	}
	u, err := c.loadPasskeyUser(ctx.GetString("user_id"))
	if err != nil {
		ctx.AbortWithStatus(500)
		return
	}
	options, data, err := engine.BeginLogin(u, webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		securityError(ctx, 400, "No passkey available for this account")
		return
	}
	c.saveChallenge(ctx, passkeyChallenge{Kind: "reauth", UserID: u.account.ID, SessionID: ctx.GetString("session_id"), Data: *data}, options)
}
func (c *Controller) FinishPasskeyReauth(ctx *gin.Context) {
	challenge, request, err := c.finishChallenge(ctx, "reauth")
	if err != nil {
		securityError(ctx, 400, "Verification expired. Try again.")
		return
	}
	engine, err := passkeyEngine()
	if err != nil {
		ctx.AbortWithStatus(503)
		return
	}
	u, err := c.loadPasskeyUser(challenge.UserID)
	if err != nil {
		ctx.AbortWithStatus(500)
		return
	}
	credential, err := engine.FinishLogin(u, challenge.Data, request)
	if err != nil || credential.Authenticator.CloneWarning {
		securityError(ctx, 401, "Passkey verification failed")
		return
	}
	if err = c.updatePasskeyCredential(u.account.ID, credential); err != nil {
		ctx.AbortWithStatus(401)
		return
	}
	c.issueSecurityGrant(ctx, true)
}
func (c *Controller) updatePasskeyCredential(userID string, credential *webauthn.Credential) error {
	encoded, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(credential.ID)
	now := time.Now().UTC()
	var existing user.Passkey
	if err := c.GormDB.Where("user_id = ? AND credential_hash = ?", userID, hex.EncodeToString(sum[:])).First(&existing).Error; err != nil {
		return err
	}
	var previous webauthn.Credential
	if err := json.Unmarshal([]byte(existing.CredentialData), &previous); err != nil {
		return err
	}
	if previous.Authenticator.SignCount != 0 && credential.Authenticator.SignCount != 0 && previous.Authenticator.SignCount >= credential.Authenticator.SignCount {
		return errors.New("passkey counter did not advance")
	}
	result := c.GormDB.Model(&user.Passkey{}).Where("id = ? AND credential_data = ?", existing.ID, existing.CredentialData).Updates(map[string]any{"credential_data": string(encoded), "last_used_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("passkey no longer registered")
	}
	return nil
}
func (c *Controller) FinishPasskeyLogin(ctx *gin.Context) {
	challenge, request, err := c.finishChallenge(ctx, "login")
	if err != nil {
		securityError(ctx, 400, "Sign-in expired. Try again.")
		return
	}
	engine, err := passkeyEngine()
	if err != nil {
		ctx.AbortWithStatus(503)
		return
	}
	var resolved *passkeyUser
	credential, err := engine.FinishDiscoverableLogin(func(rawID, handle []byte) (webauthn.User, error) {
		sum := sha256.Sum256(rawID)
		var row user.Passkey
		if e := c.GormDB.Where("credential_hash = ?", hex.EncodeToString(sum[:])).First(&row).Error; e != nil {
			return nil, e
		}
		u, e := c.loadPasskeyUser(row.UserID)
		if e != nil {
			return nil, e
		}
		if !bytes.Equal(u.WebAuthnID(), handle) {
			return nil, errors.New("invalid user handle")
		}
		resolved = u
		return u, nil
	}, challenge.Data, request)
	if err != nil || resolved == nil || credential.Authenticator.CloneWarning {
		identifier := "passkey"
		if resolved != nil {
			identifier = resolved.account.Username
		}
		c.recordPasskeyAttempt(ctx, identifier, false, "Passkey verification failed")
		securityError(ctx, 401, "Passkey sign-in failed. Try again or use another method.")
		return
	}
	ua, err := c.repo.GetUserAuthRepository().GetUserAuthByUserID(resolved.account.ID)
	if err != nil {
		ctx.AbortWithStatus(401)
		return
	}
	blocked, err := c.repo.GetUserAuthRepository().IsLoginTemporarilyBlocked(resolved.account.ID)
	if err != nil || blocked || resolved.account.IsAccountLocked() || ua.AccountStatus != user.AccountStatusActive || !ua.IsEmailVerified {
		securityError(ctx, 403, "Account is unavailable. Use account recovery or contact support.")
		return
	}
	if err = c.updatePasskeyCredential(resolved.account.ID, credential); err != nil {
		ctx.AbortWithStatus(401)
		return
	}
	if ua.HasEnabledTOTP() || isPrivilegedRole(resolved.account.Role) {
		if err := c.requireSecondFactor(ctx, resolved.account, ua, "Passkey verified"); err != nil {
			c.recordPasskeyAttempt(ctx, resolved.account.Username, false, "Passkey sign-in blocked by account policy")
		} else {
			c.recordPasskeyAttempt(ctx, resolved.account.Username, true, "Passkey verified; authenticator verification required")
		}
		return
	}
	if err := c.CompleteLogin(ctx, resolved.account, ua, user.LoginMethodPasskey, "Signed in with your passkey"); err == nil {
		c.recordPasskeyAttempt(ctx, resolved.account.Username, true, "Passkey sign-in successful")
	}
}
func (c *Controller) passkeyNotice(account *user.User, action, name string) {
	go func() {
		if err := c.emailService.SendPasskeyChangedEmail(account.Email, account.Username, action, name); err != nil {
			logger.Logger.Warn("Failed to send passkey security email", "error", err)
		}
	}()
}

// Login history includes verified intermediate factors as well as completed logins,
// matching the existing Google sign-in attempt history.
func (c *Controller) recordPasskeyAttempt(ctx *gin.Context, identifier string, success bool, reason string) {
	if err := c.repo.GetLoginAttemptRepository().RecordLoginAttempt(ctx.ClientIP(), ctx.GetHeader("User-Agent"), success, reason, identifier); err != nil {
		logger.Logger.Warn("Failed to record passkey attempt", "error", err)
	}
}
