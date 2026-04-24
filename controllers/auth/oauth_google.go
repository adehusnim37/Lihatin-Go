package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	"github.com/adehusnim37/lihatin-go/internal/pkg/disposable"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	httputil "github.com/adehusnim37/lihatin-go/internal/pkg/http"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/validator"
	"github.com/adehusnim37/lihatin-go/models/user"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	googleOAuthAuthorizeEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"
	googleOAuthTokenEndpoint     = "https://oauth2.googleapis.com/token"
	googleOAuthTokenInfoEndpoint = "https://oauth2.googleapis.com/tokeninfo"
)

var oauthUsernameSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

type googleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       string
}

type googleTokenExchangeResponse struct {
	AccessToken      string `json:"access_token"`
	IDToken          string `json:"id_token"`
	ExpiresIn        int    `json:"expires_in"`
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type googleTokenInfoResponse struct {
	Aud              string          `json:"aud"`
	Azp              string          `json:"azp"`
	Sub              string          `json:"sub"`
	Email            string          `json:"email"`
	EmailVerifiedRaw json.RawMessage `json:"email_verified"`
	ExpRaw           json.RawMessage `json:"exp"`
	Iss              string          `json:"iss"`
	Name             string          `json:"name"`
	GivenName        string          `json:"given_name"`
	FamilyName       string          `json:"family_name"`
	Picture          string          `json:"picture"`
	Error            string          `json:"error"`
	ErrorDescription string          `json:"error_description"`
}

func (g *googleTokenInfoResponse) IsEmailVerified() bool {
	var valueBool bool
	if err := json.Unmarshal(g.EmailVerifiedRaw, &valueBool); err == nil {
		return valueBool
	}

	var valueString string
	if err := json.Unmarshal(g.EmailVerifiedRaw, &valueString); err == nil {
		return strings.EqualFold(valueString, "true") || valueString == "1"
	}

	return false
}

func (g *googleTokenInfoResponse) ExpiresAtUnix() (int64, error) {
	var valueInt int64
	if err := json.Unmarshal(g.ExpRaw, &valueInt); err == nil {
		return valueInt, nil
	}

	var valueString string
	if err := json.Unmarshal(g.ExpRaw, &valueString); err == nil {
		parsed, parseErr := strconv.ParseInt(valueString, 10, 64)
		if parseErr != nil {
			return 0, parseErr
		}
		return parsed, nil
	}

	return 0, errors.New("invalid exp format")
}

func (g *googleTokenInfoResponse) IsValidIssuer() bool {
	switch strings.ToLower(strings.TrimSpace(g.Iss)) {
	case "accounts.google.com", "https://accounts.google.com":
		return true
	default:
		return false
	}
}

// StartGoogleOAuth initializes Google OAuth authorization flow.
func (c *Controller) StartGoogleOAuth(ctx *gin.Context) {
	var req dto.GoogleOAuthStartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	cfg, err := getGoogleOAuthConfig(false)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "GOOGLE_OAUTH_NOT_CONFIGURED", "Google OAuth is not configured", "auth")
		return
	}

	intent := auth.GoogleOAuthIntent(strings.ToLower(strings.TrimSpace(req.Intent)))
	state, err := auth.GenerateAndStoreGoogleOAuthState(ctx.Request.Context(), intent)
	if err != nil {
		logger.Logger.Error("Failed to generate Google OAuth state",
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "GOOGLE_OAUTH_STATE_FAILED", "Failed to start Google OAuth", "auth")
		return
	}

	authURL := buildGoogleAuthorizationURL(cfg, state)
	httputil.SendOKResponse(ctx, dto.GoogleOAuthStartResponse{
		AuthorizationURL: authURL,
		State:            state,
	}, "Google OAuth started")
}

// GoogleOAuthCallback exchanges Google auth code and continues login flow.
func (c *Controller) GoogleOAuthCallback(ctx *gin.Context) {
	var req dto.GoogleOAuthCallbackRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		validator.SendValidationError(ctx, err, &req)
		return
	}

	cfg, err := getGoogleOAuthConfig(true)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "GOOGLE_OAUTH_NOT_CONFIGURED", "Google OAuth is not configured", "auth")
		return
	}

	statePayload, err := auth.ConsumeGoogleOAuthState(ctx.Request.Context(), req.State)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrOAuthStateNotFound):
			httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "INVALID_OAUTH_STATE", "OAuth session expired or invalid. Please try again.", "state")
		case errors.Is(err, auth.ErrOAuthStateServiceUnavailable):
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "OAUTH_STATE_SERVICE_UNAVAILABLE", "OAuth state service is unavailable", "state")
		default:
			logger.Logger.Error("Failed to consume Google OAuth state",
				"error", err.Error(),
			)
			httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "OAUTH_STATE_VALIDATION_FAILED", "Failed to validate OAuth state", "state")
		}
		return
	}

	allowCreate := statePayload != nil && statePayload.Intent == auth.GoogleOAuthIntentSignup

	tokenResp, err := exchangeGoogleCode(ctx.Request.Context(), cfg, req.Code)
	if err != nil {
		logger.Logger.Warn("Google OAuth code exchange failed",
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "GOOGLE_OAUTH_EXCHANGE_FAILED", "Failed to validate Google authorization code", "code")
		return
	}

	tokenInfo, err := verifyGoogleIDToken(ctx.Request.Context(), tokenResp.IDToken)
	if err != nil {
		logger.Logger.Warn("Google OAuth token verification failed",
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "GOOGLE_OAUTH_TOKEN_INVALID", "Google identity token is invalid", "code")
		return
	}

	email := strings.ToLower(strings.TrimSpace(tokenInfo.Email))
	sub := strings.TrimSpace(tokenInfo.Sub)
	if email == "" || sub == "" || !tokenInfo.IsEmailVerified() {
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "GOOGLE_OAUTH_CLAIMS_INVALID", "Google account must have a verified email", "auth")
		return
	}

	if tokenInfo.Aud != cfg.ClientID {
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "GOOGLE_OAUTH_AUDIENCE_INVALID", "Google token audience mismatch", "auth")
		return
	}

	if !tokenInfo.IsValidIssuer() {
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "GOOGLE_OAUTH_ISSUER_INVALID", "Google token issuer is invalid", "auth")
		return
	}

	expUnix, err := tokenInfo.ExpiresAtUnix()
	if err != nil || expUnix <= time.Now().Unix() {
		httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "GOOGLE_OAUTH_TOKEN_EXPIRED", "Google token has expired", "auth")
		return
	}

	usr, userAuth, err := c.resolveGoogleOAuthIdentity(ctx.Request.Context(), email, sub, tokenInfo, allowCreate)
	if err != nil {
		if errors.Is(err, disposable.ErrDisposableEmailBlocked) {
			httputil.SendErrorResponse(
				ctx,
				http.StatusBadRequest,
				"DISPOSABLE_EMAIL_BLOCKED",
				"Disposable email addresses are not allowed. Please use a permanent email address.",
				"email",
			)
			return
		}
		if errors.Is(err, apperrors.ErrUserNotFound) {
			httputil.SendErrorResponse(ctx, http.StatusUnauthorized, "ACCOUNT_NOT_REGISTERED", "Account not found. Please sign up first.", "auth")
			return
		}
		logger.Logger.Error("Failed to resolve Google OAuth identity",
			"email", email,
			"error", err.Error(),
		)
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "GOOGLE_OAUTH_LOGIN_FAILED", "Failed to process Google login", "auth")
		return
	}

	if usr.IsLocked {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "USER_LOCKED", "Your account has been locked. Please contact support.", "auth")
		return
	}

	isLocked, err := c.repo.GetUserAuthRepository().IsAccountLocked(usr.ID)
	if err != nil {
		httputil.SendErrorResponse(ctx, http.StatusInternalServerError, "LOGIN_FAILED", "An error occurred during login", "auth")
		return
	}
	if isLocked {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "ACCOUNT_LOCKED", "Your account is locked. Please try again later.", "auth")
		return
	}
	if !userAuth.IsActive {
		httputil.SendErrorResponse(ctx, http.StatusForbidden, "ACCOUNT_DEACTIVATED", "Your account has been deactivated. Please contact support.", "auth")
		return
	}

	if err := c.requireSecondFactor(ctx, usr, userAuth, "Google sign-in verified"); err != nil {
		return
	}
}

func getGoogleOAuthConfig(requireSecret bool) (googleOAuthConfig, error) {
	cfg := googleOAuthConfig{
		ClientID:    strings.TrimSpace(config.GetEnvOrDefault(config.EnvGoogleOAuthClientID, "")),
		RedirectURI: strings.TrimSpace(config.GetEnvOrDefault(config.EnvGoogleOAuthRedirectURI, "")),
		Scopes:      strings.TrimSpace(config.GetEnvOrDefault(config.EnvGoogleOAuthScopes, "openid email profile")),
	}

	if cfg.ClientID == "" || cfg.RedirectURI == "" {
		return googleOAuthConfig{}, errors.New("google oauth client_id or redirect_uri is missing")
	}

	if cfg.Scopes == "" {
		cfg.Scopes = "openid email profile"
	}

	if requireSecret {
		cfg.ClientSecret = strings.TrimSpace(config.GetEnvOrDefault(config.EnvGoogleOAuthClientSecret, ""))
		if cfg.ClientSecret == "" {
			return googleOAuthConfig{}, errors.New("google oauth client_secret is missing")
		}
	}

	return cfg, nil
}

func buildGoogleAuthorizationURL(cfg googleOAuthConfig, state string) string {
	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", cfg.ClientID)
	values.Set("redirect_uri", cfg.RedirectURI)
	values.Set("scope", cfg.Scopes)
	values.Set("state", state)
	values.Set("prompt", "select_account")
	values.Set("access_type", "online")
	values.Set("include_granted_scopes", "true")
	return googleOAuthAuthorizeEndpoint + "?" + values.Encode()
}

func exchangeGoogleCode(ctx context.Context, cfg googleOAuthConfig, code string) (*googleTokenExchangeResponse, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("redirect_uri", cfg.RedirectURI)
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleOAuthTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp googleTokenExchangeResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		reason := strings.TrimSpace(tokenResp.ErrorDescription)
		if reason == "" {
			reason = strings.TrimSpace(tokenResp.Error)
		}
		if reason == "" {
			reason = "unknown error"
		}
		return nil, fmt.Errorf("google token exchange failed: %s", reason)
	}

	if strings.TrimSpace(tokenResp.IDToken) == "" {
		return nil, errors.New("missing id_token in google response")
	}

	return &tokenResp, nil
}

func verifyGoogleIDToken(ctx context.Context, idToken string) (*googleTokenInfoResponse, error) {
	endpoint := googleOAuthTokenInfoEndpoint + "?id_token=" + url.QueryEscape(idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenInfo googleTokenInfoResponse
	if err := json.Unmarshal(body, &tokenInfo); err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		reason := strings.TrimSpace(tokenInfo.ErrorDescription)
		if reason == "" {
			reason = strings.TrimSpace(tokenInfo.Error)
		}
		if reason == "" {
			reason = "tokeninfo rejected id_token"
		}
		return nil, fmt.Errorf("google tokeninfo verification failed: %s", reason)
	}

	return &tokenInfo, nil
}

func (c *Controller) resolveGoogleOAuthIdentity(ctx context.Context, email, providerUserID string, tokenInfo *googleTokenInfoResponse, allowCreate bool) (*user.User, *user.UserAuth, error) {
	metadata := buildGoogleAuthMetadata(email, tokenInfo)

	// 1) Primary lookup by existing provider mapping.
	existingMethod, err := c.repo.GetAuthMethodRepository().GetAuthMethodByProviderUserID(user.AuthMethodTypeOAuthGoogle, providerUserID)
	if err == nil {
		userAuth, uaErr := c.repo.GetUserAuthRepository().GetUserAuthByID(existingMethod.UserAuthID)
		if uaErr != nil {
			return nil, nil, uaErr
		}

		usr, userErr := c.repo.GetUserRepository().GetUserByID(userAuth.UserID)
		if userErr != nil {
			return nil, nil, userErr
		}

		if !userAuth.IsEmailVerified {
			userAuth.IsEmailVerified = true
			if saveErr := c.repo.GetUserAuthRepository().UpdateUserAuth(userAuth); saveErr != nil {
				return nil, nil, saveErr
			}
		}

		if upsertErr := c.upsertGoogleAuthMethod(userAuth.ID, providerUserID, metadata); upsertErr != nil {
			return nil, nil, upsertErr
		}

		return usr, userAuth, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}

	// 2) Auto-link by email when existing account found.
	usr, userAuth, err := c.repo.GetUserAuthRepository().GetUserForLogin(email)
	if err == nil {
		if !userAuth.IsEmailVerified {
			userAuth.IsEmailVerified = true
			if saveErr := c.repo.GetUserAuthRepository().UpdateUserAuth(userAuth); saveErr != nil {
				return nil, nil, saveErr
			}
		}

		if upsertErr := c.upsertGoogleAuthMethod(userAuth.ID, providerUserID, metadata); upsertErr != nil {
			return nil, nil, upsertErr
		}
		return usr, userAuth, nil
	}
	if !errors.Is(err, apperrors.ErrUserNotFound) {
		return nil, nil, err
	}

	if !allowCreate {
		return nil, nil, apperrors.ErrUserNotFound
	}

	if policy := disposable.Global(); policy != nil {
		blocked, policyErr := policy.ShouldBlockEmail(ctx, email)
		if policyErr != nil {
			logger.Logger.Warn("Disposable email policy check failed for Google OAuth signup",
				"email", email,
				"error", policyErr.Error(),
			)
		}
		if blocked {
			return nil, nil, disposable.ErrDisposableEmailBlocked
		}
	}

	// 3) Create new OAuth user if no existing account.
	createdUser, createdUserAuth, createErr := c.createGoogleOAuthUser(ctx, email, providerUserID, tokenInfo, metadata)
	if createErr != nil {
		return nil, nil, createErr
	}

	return createdUser, createdUserAuth, nil
}

func (c *Controller) createGoogleOAuthUser(ctx context.Context, email, providerUserID string, tokenInfo *googleTokenInfoResponse, metadata string) (*user.User, *user.UserAuth, error) {
	firstName, lastName := deriveGoogleProfileNames(tokenInfo)
	placeholderPassword, err := generateOAuthPlaceholderPassword()
	if err != nil {
		return nil, nil, err
	}

	createdUser := &user.User{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Password:  placeholderPassword,
		Avatar:    strings.TrimSpace(tokenInfo.Picture),
	}

	var userAuth *user.UserAuth
	created := false
	for attempt := 0; attempt < 6; attempt++ {
		username, usernameErr := c.generateGoogleUsername(email)
		if usernameErr != nil {
			return nil, nil, usernameErr
		}
		createdUser.Username = username

		createErr := c.repo.GetUserRepository().CreateUser(createdUser)
		if createErr == nil {
			created = true
			break
		}

		if !errors.Is(createErr, apperrors.ErrUserDuplicateEntry) {
			return nil, nil, createErr
		}

		existingUser, existingAuth, lookupErr := c.repo.GetUserAuthRepository().GetUserForLogin(email)
		if lookupErr == nil {
			if !existingAuth.IsEmailVerified {
				existingAuth.IsEmailVerified = true
				if saveErr := c.repo.GetUserAuthRepository().UpdateUserAuth(existingAuth); saveErr != nil {
					return nil, nil, saveErr
				}
			}

			if upsertErr := c.upsertGoogleAuthMethod(existingAuth.ID, providerUserID, metadata); upsertErr != nil {
				return nil, nil, upsertErr
			}
			return existingUser, existingAuth, nil
		}
	}

	if !created {
		return nil, nil, errors.New("failed to create google oauth user")
	}

	hashedPassword, err := auth.HashPassword(placeholderPassword)
	if err != nil {
		return nil, nil, err
	}

	authUUID, err := uuid.NewV7()
	if err != nil {
		return nil, nil, err
	}

	userAuth = &user.UserAuth{
		ID:              authUUID.String(),
		UserID:          createdUser.ID,
		PasswordHash:    hashedPassword,
		IsEmailVerified: true,
		IsActive:        true,
	}

	if err := c.repo.GetUserAuthRepository().CreateUserAuth(userAuth); err != nil {
		existingAuth, lookupErr := c.repo.GetUserAuthRepository().GetUserAuthByUserID(createdUser.ID)
		if lookupErr != nil {
			return nil, nil, err
		}
		userAuth = existingAuth
	}

	if upsertErr := c.upsertGoogleAuthMethod(userAuth.ID, providerUserID, metadata); upsertErr != nil {
		return nil, nil, upsertErr
	}

	logger.Logger.Info("Created new user via Google OAuth",
		"user_id", createdUser.ID,
		"email", createdUser.Email,
	)

	return createdUser, userAuth, nil
}

func (c *Controller) upsertGoogleAuthMethod(userAuthID, providerUserID, metadata string) error {
	repo := c.repo.GetAuthMethodRepository()
	now := time.Now()

	existing, err := repo.GetAuthMethodByType(userAuthID, user.AuthMethodTypeOAuthGoogle)
	if err == nil {
		existing.ProviderUserID = providerUserID
		existing.Metadata = metadata
		existing.FriendlyName = "Google OAuth"
		existing.IsEnabled = true
		existing.IsVerified = true
		existing.VerifiedAt = &now
		existing.DisabledAt = nil

		if updateErr := repo.UpdateAuthMethod(existing); updateErr != nil {
			return updateErr
		}
		_ = repo.UpdateLastUsed(existing.ID)
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	methodID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	method := &user.AuthMethod{
		ID:             methodID.String(),
		UserAuthID:     userAuthID,
		Type:           user.AuthMethodTypeOAuthGoogle,
		IsEnabled:      true,
		IsVerified:     true,
		VerifiedAt:     &now,
		LastUsedAt:     &now,
		FriendlyName:   "Google OAuth",
		ProviderUserID: providerUserID,
		Metadata:       metadata,
	}

	return repo.CreateAuthMethod(method)
}

func (c *Controller) generateGoogleUsername(email string) (string, error) {
	base := sanitizeGoogleUsernameBase(email)
	if len(base) > 24 {
		base = base[:24]
	}

	for attempt := 0; attempt < 20; attempt++ {
		candidate := base
		if attempt > 0 {
			suffix, err := auth.GenerateSecureToken(2)
			if err != nil {
				return "", err
			}
			maxBaseLength := 30 - len(suffix) - 1
			if maxBaseLength < 3 {
				maxBaseLength = 3
			}
			if len(candidate) > maxBaseLength {
				candidate = candidate[:maxBaseLength]
			}
			candidate = candidate + "_" + suffix
		}

		_, _, err := c.repo.GetUserAuthRepository().GetUserForLogin(candidate)
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}

	finalSuffix, err := auth.GenerateSecureToken(4)
	if err != nil {
		return "", err
	}

	maxBaseLength := 30 - len(finalSuffix) - 1
	if maxBaseLength < 3 {
		maxBaseLength = 3
	}
	if len(base) > maxBaseLength {
		base = base[:maxBaseLength]
	}

	return base + "_" + finalSuffix, nil
}

func sanitizeGoogleUsernameBase(email string) string {
	localPart := strings.TrimSpace(strings.Split(email, "@")[0])
	localPart = strings.ToLower(localPart)
	localPart = oauthUsernameSanitizer.ReplaceAllString(localPart, "")
	if localPart == "" {
		localPart = "user"
	}
	if len(localPart) < 3 {
		localPart = localPart + "user"
		if len(localPart) > 30 {
			localPart = localPart[:30]
		}
	}
	return localPart
}

func deriveGoogleProfileNames(tokenInfo *googleTokenInfoResponse) (string, string) {
	firstName := normalizeProfileNamePart(tokenInfo.GivenName, "Google")
	lastName := normalizeProfileNamePart(tokenInfo.FamilyName, "User")

	if tokenInfo.Name != "" {
		parts := strings.Fields(strings.TrimSpace(tokenInfo.Name))
		if len(parts) > 0 {
			if strings.TrimSpace(tokenInfo.GivenName) == "" {
				firstName = normalizeProfileNamePart(parts[0], "Google")
			}
			if strings.TrimSpace(tokenInfo.FamilyName) == "" {
				if len(parts) > 1 {
					lastName = normalizeProfileNamePart(strings.Join(parts[1:], " "), "User")
				}
			}
		}
	}

	return firstName, lastName
}

func normalizeProfileNamePart(value, fallback string) string {
	name := strings.TrimSpace(value)
	if name == "" {
		name = fallback
	}
	if len(name) < 3 {
		name = fallback
	}
	if len(name) > 50 {
		name = name[:50]
	}
	return name
}

func generateOAuthPlaceholderPassword() (string, error) {
	randomToken, err := auth.GenerateSecureToken(20)
	if err != nil {
		return "", err
	}
	return "Gg1!" + randomToken, nil
}

func buildGoogleAuthMetadata(email string, tokenInfo *googleTokenInfoResponse) string {
	payload := map[string]string{
		"provider": "google",
		"email":    email,
		"issuer":   strings.TrimSpace(tokenInfo.Iss),
		"azp":      strings.TrimSpace(tokenInfo.Azp),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
