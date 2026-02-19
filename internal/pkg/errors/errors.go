package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application error with HTTP metadata
type AppError struct {
	Code       string // Error code (e.g., "USER_NOT_FOUND")
	Message    string // User-friendly message
	StatusCode int    // HTTP status code
	Field      string // Field name for validation errors
	Err        error  // Original error (optional)
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new AppError
func NewAppError(code, message string, statusCode int, field string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Field:      field,
	}
}

// WithError wraps an existing error
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// ============================================
// Pre-defined Application Errors
// ============================================

// User Errors
var (
	ErrUserNotFound = NewAppError(
		"USER_NOT_FOUND",
		"User not found",
		http.StatusNotFound,
		"user",
	)
	ErrUserAlreadyExists = NewAppError(
		"USER_EXISTS",
		"User already exists",
		http.StatusConflict,
		"user",
	)
	ErrUserEmailExists = NewAppError(
		"EMAIL_EXISTS",
		"Email already exists",
		http.StatusConflict,
		"email",
	)
	ErrUserEmailSameAsCurrent = NewAppError(
		"EMAIL_SAME_AS_CURRENT",
		"New email cannot be the same as the current email",
		http.StatusBadRequest,
		"email",
	)
	ErrEmailChangeRateLimitExceeded = NewAppError(
		"EMAIL_CHANGE_RATE_LIMIT_EXCEEDED",
		"You can only change your email once every 30 days. Please try again later.",
		http.StatusTooManyRequests,
		"email",
	)
	ErrEmailChangeLocked = NewAppError(
		"EMAIL_CHANGE_LOCKED",
		"Email change is temporarily locked due to suspicious activity. Please contact support.",
		http.StatusForbidden,
		"email",
	)
	ErrUserUsernameExists = NewAppError(
		"USERNAME_EXISTS",
		"Username already exists",
		http.StatusConflict,
		"username",
	)
	ErrUsernameChangeNotAllowed = NewAppError(
		"USERNAME_CHANGE_NOT_ALLOWED",
		"Username can only be changed once and cannot be changed again",
		http.StatusBadRequest,
		"username",
	)
	ErrDuplicateUsername = NewAppError(
		"USERNAME_EXISTS",
		"Username already exists",
		http.StatusConflict,
		"username",
	)
	ErrUserCreationFailed = NewAppError(
		"USER_CREATE_FAILED",
		"Failed to create user",
		http.StatusBadRequest,
		"user",
	)
	ErrUserUpdateFailed = NewAppError(
		"USER_UPDATE_FAILED",
		"Failed to update user",
		http.StatusBadRequest,
		"user",
	)
	ErrUserDeleteFailed = NewAppError(
		"USER_DELETE_FAILED",
		"Failed to delete user",
		http.StatusBadRequest,
		"user",
	)
	ErrUserLockFailed = NewAppError(
		"USER_LOCK_FAILED",
		"Failed to lock user",
		http.StatusBadRequest,
		"user",
	)
	ErrUserUnlockFailed = NewAppError(
		"USER_UNLOCK_FAILED",
		"Failed to unlock user",
		http.StatusBadRequest,
		"user",
	)
	ErrUserAlreadyLocked = NewAppError(
		"USER_ALREADY_LOCKED",
		"User is already locked",
		http.StatusBadRequest,
		"user",
	)
	ErrUserNotLocked = NewAppError(
		"USER_NOT_LOCKED",
		"User is not locked",
		http.StatusBadRequest,
		"user",
	)
	ErrUserInvalidCredentials = NewAppError(
		"INVALID_CREDENTIALS",
		"Invalid email/username or password",
		http.StatusUnauthorized,
		"auth",
	)
	ErrUserAccountLocked = NewAppError(
		"ACCOUNT_LOCKED",
		"User account is locked",
		http.StatusForbidden,
		"user",
	)
	ErrUserAccountDeleted = NewAppError(
		"ACCOUNT_DELETED",
		"User account has been deleted",
		http.StatusForbidden,
		"user",
	)
	ErrUserPasswordHashFailed = NewAppError(
		"PASSWORD_HASH_FAILED",
		"Failed to hash user password",
		http.StatusInternalServerError,
		"password",
	)
	ErrUserInvalidInput = NewAppError(
		"INVALID_INPUT",
		"Invalid user input data",
		http.StatusBadRequest,
		"input",
	)
	ErrUserUnauthorized = NewAppError(
		"UNAUTHORIZED",
		"User is not authorized to perform this action",
		http.StatusForbidden,
		"auth",
	)
	ErrUserAccountDeactivated = NewAppError(
		"USER_ACCOUNT_DEACTIVATED",
		"User account has been deactivated",
		http.StatusForbidden,
		"user",
	)
	ErrUserValidationFailed = NewAppError(
		"VALIDATION_FAILED",
		"User data validation failed",
		http.StatusBadRequest,
		"validation",
	)
	ErrUserDuplicateEntry = NewAppError(
		"DUPLICATE_ENTRY",
		"Duplicate entry for user data",
		http.StatusConflict,
		"user",
	)
	ErrUserDatabaseError = NewAppError(
		"DATABASE_ERROR",
		"Database error while processing user request",
		http.StatusInternalServerError,
		"database",
	)
	ErrUserEmailNotVerified = NewAppError(
		"EMAIL_NOT_VERIFIED",
		"User email is not verified",
		http.StatusForbidden,
		"email",
	)
	ErrUserFindFailed = NewAppError(
		"USER_FIND_FAILED",
		"Failed to find user",
		http.StatusBadRequest,
		"user",
	)
	ErrUserFailed = NewAppError(
		"USER_FAILED",
		"Failed to process user request",
		http.StatusBadRequest,
		"user",
	)
)

// User Auth Errors
var (
	ErrInvalidCredentials = NewAppError(
		"INVALID_CREDENTIALS",
		"Invalid email/username or password",
		http.StatusUnauthorized,
		"auth",
	)
	ErrUserAuthNotFound = NewAppError(
		"USER_AUTH_NOT_FOUND",
		"User authentication data not found",
		http.StatusNotFound,
		"user_auth",
	)
	ErrUserAuthFindFailed = NewAppError(
		"USER_AUTH_FIND_FAILED",
		"Failed to find user authentication data",
		http.StatusBadRequest,
		"user_auth",
	)
	ErrUserAuthEmailNotVerified = NewAppError(
		"USER_AUTH_EMAIL_NOT_VERIFIED",
		"User email is not verified",
		http.StatusForbidden,
		"email",
	)
	ErrUserAuthUpdateFailed = NewAppError(
		"USER_AUTH_UPDATE_FAILED",
		"Failed to update user authentication data",
		http.StatusBadRequest,
		"user_auth",
	)
	ErrUserAuthCreateFailed = NewAppError(
		"USER_AUTH_CREATE_FAILED",
		"Failed to create user authentication data",
		http.StatusBadRequest,
		"user_auth",
	)
	ErrUserAuthDeleteFailed = NewAppError(
		"USER_AUTH_DELETE_FAILED",
		"Failed to delete user authentication data",
		http.StatusBadRequest,
		"user_auth",
	)
	ErrUserAuthLockFailed = NewAppError(
		"USER_AUTH_LOCK_FAILED",
		"Failed to lock user authentication data",
		http.StatusBadRequest,
		"user_auth",
	)
	ErrUserAuthPasswordResetFailed = NewAppError(
		"USER_AUTH_PASSWORD_RESET_FAILED",
		"Failed to reset user password",
		http.StatusBadRequest,
		"user_auth",
	)
	ErrUserAuthExpiredOrInvalidToken = NewAppError(
		"USER_AUTH_EXPIRED_OR_INVALID_TOKEN",
		"User authentication token is expired or invalid",
		http.StatusBadRequest,
		"user_auth",
	)
)

// Short Link Errors
var (
	ErrShortCreatedFailed = NewAppError(
		"SHORT_CREATE_FAILED",
		"Failed to create short link",
		http.StatusBadRequest,
		"short_link",
	)
)

// Short Link Detail Errors
var (
	ErrShortDetailNotFound = NewAppError(
		"SHORT_DETAIL_NOT_FOUND",
		"Short link detail not found",
		http.StatusNotFound,
		"short_detail",
	)
	ErrShortDetailFindFailed = NewAppError(
		"SHORT_DETAIL_FIND_FAILED",
		"Failed to find short link detail",
		http.StatusBadRequest,
		"short_detail",
	)
	ErrShortDetailUpdateFailed = NewAppError(
		"SHORT_DETAIL_UPDATE_FAILED",
		"Failed to update short link detail",
		http.StatusBadRequest,
		"short_detail",
	)
	ErrShortDetailCreatedFailed = NewAppError(
		"SHORT_DETAIL_CREATE_FAILED",
		"Failed to create short link detail",
		http.StatusBadRequest,
		"short_detail",
	)
	ErrShortUpdateFailed = NewAppError(
		"SHORT_UPDATE_FAILED",
		"Failed to update short link",
		http.StatusBadRequest,
		"short_link",
	)
	ErrShortDeleteFailed = NewAppError(
		"SHORT_DELETE_FAILED",
		"Failed to delete short link",
		http.StatusBadRequest,
		"short_link",
	)
	ErrShortCheckFailed = NewAppError(
		"SHORT_CHECK_FAILED",
		"Failed to check short link",
		http.StatusInternalServerError,
		"short_link",
	)
	ErrShortListFailed = NewAppError(
		"SHORT_LIST_FAILED",
		"Failed to list short links",
		http.StatusInternalServerError,
		"short_link",
	)
	ErrShortBanFailed = NewAppError(
		"SHORT_BAN_FAILED",
		"Failed to ban short link",
		http.StatusBadRequest,
		"short_link",
	)
	ErrShortRestoreFailed = NewAppError(
		"SHORT_RESTORE_FAILED",
		"Failed to restore short link",
		http.StatusBadRequest,
		"short_link",
	)
	ErrShortResetPasscodeFailed = NewAppError(
		"SHORT_RESET_PASSCODE_FAILED",
		"Failed to reset passcode",
		http.StatusBadRequest,
		"short_link",
	)
	ErrShortValidateTokenFailed = NewAppError(
		"SHORT_VALIDATE_TOKEN_FAILED",
		"Failed to validate token",
		http.StatusBadRequest,
		"token",
	)
	ErrShortStatsFailed = NewAppError(
		"SHORT_STATS_FAILED",
		"Failed to get short link stats",
		http.StatusInternalServerError,
		"short_link",
	)
	ErrShortBulkCreateFailed = NewAppError(
		"SHORT_BULK_CREATE_FAILED",
		"Failed to create bulk short links",
		http.StatusBadRequest,
		"short_link",
	)
	ErrShortViewTrackFailed = NewAppError(
		"SHORT_VIEW_TRACK_FAILED",
		"Failed to track short link view",
		http.StatusInternalServerError,
		"short_link",
	)
	ErrShortGetFailed = NewAppError(
		"SHORT_GET_FAILED",
		"Failed to get short link",
		http.StatusBadRequest,
		"short_link",
	)
	ErrShortLinkNotFound = NewAppError(
		"SHORT_LINK_NOT_FOUND",
		"Short link not found",
		http.StatusNotFound,
		"short_link",
	)
	ErrShortLinkExpired = NewAppError(
		"SHORT_LINK_EXPIRED",
		"Short link has expired",
		http.StatusGone,
		"short_link",
	)
	ErrShortLinkInactive = NewAppError(
		"SHORT_LINK_INACTIVE",
		"Short link is inactive",
		http.StatusForbidden,
		"short_link",
	)
	ErrShortLinkUnauthorized = NewAppError(
		"SHORT_LINK_UNAUTHORIZED",
		"Unauthorized to access this link",
		http.StatusForbidden,
		"short_link",
	)
	ErrDuplicateShortCode = NewAppError(
		"DUPLICATE_SHORT_CODE",
		"Short code already exists",
		http.StatusConflict,
		"short_code",
	)
	ErrInvalidOriginalURL = NewAppError(
		"INVALID_ORIGINAL_URL",
		"Invalid original URL",
		http.StatusBadRequest,
		"original_url",
	)
	ErrShortLinkAlreadyDeleted = NewAppError(
		"SHORT_LINK_ALREADY_DELETED",
		"Short link has already been deleted",
		http.StatusBadRequest,
		"short_link",
	)
	ErrEmptyCodesList = NewAppError(
		"EMPTY_CODES_LIST",
		"Codes list cannot be empty",
		http.StatusBadRequest,
		"codes",
	)
	ErrSomeShortLinksNotFound = NewAppError(
		"SOME_SHORT_LINKS_NOT_FOUND",
		"Some short links not found",
		http.StatusNotFound,
		"short_links",
	)
	ErrLinkIsBanned = NewAppError(
		"LINK_BANNED",
		"The original URL is banned",
		http.StatusForbidden,
		"original_url",
	)
	ErrShortIsNotDeleted = NewAppError(
		"SHORT_NOT_DELETED",
		"Short link is not deleted",
		http.StatusBadRequest,
		"short_link",
	)
	ErrFailedToGenerateCode = NewAppError(
		"CODE_GENERATION_FAILED",
		"Failed to generate unique short code",
		http.StatusInternalServerError,
		"short_code",
	)
	ErrInvalidPasscode = NewAppError(
		"INVALID_PASSCODE",
		"Invalid passcode format",
		http.StatusBadRequest,
		"passcode",
	)
	ErrInvalidClickLimit = NewAppError(
		"INVALID_CLICK_LIMIT",
		"Click limit must be a positive integer",
		http.StatusBadRequest,
		"click_limit",
	)
	ErrStatsNotEnabled = NewAppError(
		"STATS_NOT_ENABLED",
		"Statistics tracking is not enabled for this link",
		http.StatusBadRequest,
		"short_link",
	)
	ErrPasscodeRequired = NewAppError(
		"PASSCODE_REQUIRED",
		"Passcode required",
		http.StatusUnauthorized,
		"passcode",
	)
	ErrPasscodeIncorrect = NewAppError(
		"PASSCODE_INCORRECT",
		"Incorrect passcode",
		http.StatusUnauthorized,
		"passcode",
	)
	ErrClickLimitReached = NewAppError(
		"CLICK_LIMIT_REACHED",
		"Click limit reached",
		http.StatusForbidden,
		"short_link",
	)
	ErrBulkCreateLimitExceeded = NewAppError(
		"BULK_CREATE_LIMIT_EXCEEDED",
		"Bulk create limit exceeded, maximum 15 links per batch",
		http.StatusBadRequest,
		"links",
	)
	ErrBulkCreateFailed = NewAppError(
		"BULK_CREATE_FAILED",
		"Failed to create bulk short links",
		http.StatusBadRequest,
		"links",
	)
	ErrEmptyBulkLinksList = NewAppError(
		"EMPTY_BULK_LINKS_LIST",
		"Bulk links list cannot be empty",
		http.StatusBadRequest,
		"links",
	)
	ErrDuplicateShortCodeInBatch = NewAppError(
		"DUPLICATE_SHORT_CODE_IN_BATCH",
		"Duplicate short code in batch creation",
		http.StatusBadRequest,
		"short_code",
	)
)

// API Key Errors
var (
	ErrAPIKeyFailedFetching = NewAppError(
		"API_KEY_FETCH_FAILED",
		"Failed to fetch API key",
		http.StatusNotFound,
		"api_key",
	)
	ErrAPIKeyNotFound = NewAppError(
		"API_KEY_NOT_FOUND",
		"API key not found",
		http.StatusNotFound,
		"api_key",
	)
	ErrAPIKeyRevoked = NewAppError(
		"API_KEY_REVOKED",
		"API key has been revoked",
		http.StatusForbidden,
		"api_key",
	)
	ErrAPIKeyUnauthorized = NewAppError(
		"API_KEY_UNAUTHORIZED",
		"Unauthorized to access this API key",
		http.StatusUnauthorized,
		"api_key",
	)
	ErrAPIKeyNameExists = NewAppError(
		"API_KEY_NAME_EXISTS",
		"API key with the same name already exists",
		http.StatusConflict,
		"name",
	)
	ErrAPIKeyInvalidFormat = NewAppError(
		"API_KEY_INVALID_FORMAT",
		"Invalid API key format",
		http.StatusBadRequest,
		"api_key",
	)
	ErrAPIKeyInsufficientPerm = NewAppError(
		"API_KEY_INSUFFICIENT_PERMISSIONS",
		"Insufficient permissions for this API key",
		http.StatusForbidden,
		"permissions",
	)
	ErrAPIKeyMissing = NewAppError(
		"API_KEY_MISSING",
		"API key is missing",
		http.StatusBadRequest,
		"api_key",
	)
	ErrAPIKeyInactive = NewAppError(
		"API_KEY_INACTIVE",
		"API key is inactive",
		http.StatusForbidden,
		"api_key",
	)
	ErrAPIKeyInvalidIP = NewAppError(
		"API_KEY_INVALID_IP",
		"API key cannot be used from this IP address",
		http.StatusForbidden,
		"api_key",
	)
	ErrAPIKeyInvalidReferrer = NewAppError(
		"API_KEY_INVALID_REFERRER",
		"API key cannot be used from this referrer",
		http.StatusForbidden,
		"api_key",
	)
	ErrAPIKeyExpired = NewAppError(
		"API_KEY_EXPIRED",
		"API key has expired",
		http.StatusForbidden,
		"api_key",
	)
	ErrAPIKeyCreateFailed = NewAppError(
		"API_KEY_CREATE_FAILED",
		"Failed to create API key",
		http.StatusBadRequest,
		"api_key",
	)
	ErrAPIKeyUpdateFailed = NewAppError(
		"API_KEY_UPDATE_FAILED",
		"Failed to update API key",
		http.StatusBadRequest,
		"api_key",
	)
	ErrAPIKeyIDFailedFormat = NewAppError(
		"API_KEY_ID_INVALID_FORMAT",
		"Invalid API key ID format",
		http.StatusBadRequest,
		"api_key_id",
	)
	ErrAPIKeyLimitReached = NewAppError(
		"API_KEY_LIMIT_REACHED",
		"API key limit reached for user",
		http.StatusForbidden,
		"api_key",
	)
	ErrAPIKeyRateLimitExceeded = NewAppError(
		"API_KEY_RATE_LIMIT_EXCEEDED",
		"API key rate limit exceeded",
		http.StatusTooManyRequests,
		"api_key",
	)
	ErrAPIKeyIPNotAllowed = NewAppError(
		"API_KEY_IP_NOT_ALLOWED",
		"API key cannot be used from this IP address",
		http.StatusForbidden,
		"api_key",
	)
	ErrGenerateAPIKeyFailed = NewAppError(
		"GENERATE_API_KEY_FAILED",
		"Failed to generate API key",
		http.StatusInternalServerError,
		"api_key",
	)
	ErrEmptyIPAddress = NewAppError(
		"EMPTY_IP_ADDRESS",
		"Empty IP address not allowed",
		http.StatusBadRequest,
		"ip_address",
	)
	ErrAPIKeyValidationFailed = NewAppError(
		"API_KEY_VALIDATION_FAILED",
		"Failed to validate API key",
		http.StatusInternalServerError,
		"api_key",
	)
	ErrAPIKeyPermissionCheckFailed = NewAppError(
		"API_KEY_PERMISSION_CHECK_FAILED",
		"Failed to check API key permissions",
		http.StatusInternalServerError,
		"api_key",
	)
	ErrAPIKeyRevokeFailed = NewAppError(
		"API_KEY_REVOKE_FAILED",
		"Failed to revoke API key",
		http.StatusInternalServerError,
		"api_key",
	)
	ErrAPIKeyDeleteExpiredFailed = NewAppError(
		"API_KEY_DELETE_EXPIRED_FAILED",
		"Failed to delete expired API keys",
		http.StatusInternalServerError,
		"api_key",
	)
	ErrAPIKeyStatsFailed = NewAppError(
		"API_KEY_STATS_FAILED",
		"Failed to retrieve API key statistics",
		http.StatusInternalServerError,
		"api_key",
	)
	ErrAPIKeyActivateFailed = NewAppError(
		"API_KEY_ACTIVATE_FAILED",
		"Failed to activate API key",
		http.StatusInternalServerError,
		"api_key",
	)
	ErrAPIKeyDeactivateFailed = NewAppError(
		"API_KEY_DEACTIVATE_FAILED",
		"Failed to deactivate API key",
		http.StatusInternalServerError,
		"api_key",
	)
	ErrAPIKeyExpiresAtInvalid = NewAppError(
		"API_KEY_EXPIRES_AT_INVALID",
		"Expiration date must be in the future",
		http.StatusBadRequest,
		"expires_at",
	)
	ErrAPIKeyInvalidPermission = NewAppError(
		"API_KEY_INVALID_PERMISSION",
		"Invalid permission specified for API key",
		http.StatusBadRequest,
		"permissions",
	)
	ErrAPIKeyInvalidLimitUsage = NewAppError(
		"API_KEY_INVALID_LIMIT_USAGE",
		"Limit usage must be non-negative or null for unlimited",
		http.StatusBadRequest,
		"limit_usage",
	)
	ErrAPIKeyReloadFailed = NewAppError(
		"API_KEY_RELOAD_FAILED",
		"Failed to reload updated API key",
		http.StatusInternalServerError,
		"api_key",
	)
	ErrAPIKeyAlreadyActive = NewAppError(
		"API_KEY_ALREADY_ACTIVE",
		"API key is already active",
		http.StatusBadRequest,
		"api_key",
	)
)

// Activity Log Errors
var (
	ErrActivityLogNotFound = NewAppError(
		"ACTIVITY_LOG_NOT_FOUND",
		"Activity log not found",
		http.StatusNotFound,
		"activity_log",
	)
	ErrActivityLogFailed = NewAppError(
		"ACTIVITY_LOG_FAILED",
		"Failed to process activity log",
		http.StatusBadRequest,
		"activity_log",
	)
)

// login Attempt Errors
var (
	ErrLoginAttemptNotFound = NewAppError(
		"LOGIN_ATTEMPT_NOT_FOUND",
		"Login attempt not found",
		http.StatusNotFound,
		"login_attempt",
	)
	ErrLoginAttemptFailed = NewAppError(
		"LOGIN_ATTEMPT_FAILED",
		"Failed to process login attempt",
		http.StatusBadRequest,
		"login_attempt",
	)
	ErrCreateLoginAttemptFailed = NewAppError(
		"LOGIN_ATTEMPT_CREATE_FAILED",
		"Failed to create login attempt",
		http.StatusBadRequest,
		"login_attempt",
	)
)

// Email Verification Errors
var (
	ErrEmailVerificationTokenInvalidOrExpired = NewAppError(
		"INVALID_OR_EXPIRED_VERIFICATION_TOKEN",
		"Invalid or expired email verification token",
		http.StatusBadRequest,
		"token",
	)
	ErrEmailVerificationFailed = NewAppError(
		"EMAIL_VERIFICATION_FAILED",
		"Failed to verify email",
		http.StatusBadRequest,
		"email",
	)
	ErrEmailVerificationTokenInvalid = NewAppError(
		"INVALID_VERIFICATION_TOKEN",
		"Invalid email verification token",
		http.StatusBadRequest,
		"token",
	)
	ErrEmailVerificationTokenExpired = NewAppError(
		"VERIFICATION_TOKEN_EXPIRED",
		"Email verification token has expired",
		http.StatusBadRequest,
		"token",
	)
	ErrEmailAlreadyVerified = NewAppError(
		"EMAIL_ALREADY_VERIFIED",
		"Email is already verified",
		http.StatusBadRequest,
		"email",
	)
	ErrCreateVerificationTokenFailed = NewAppError(
		"CREATE_VERIFICATION_TOKEN_FAILED",
		"Failed to create email verification token",
		http.StatusBadRequest,
		"token",
	)
)

// Token Errors
var (
	ErrRevokeTokenExpired = NewAppError(
		"REVOKE_TOKEN_EXPIRED",
		"Revoke token has expired",
		http.StatusBadRequest,
		"token",
	)
	ErrRevokeTokenNotFound = NewAppError(
		"REVOKE_TOKEN_NOT_FOUND",
		"Revoke token not found",
		http.StatusBadRequest,
		"token",
	)
	ErrTokenGenerationFailed = NewAppError(
		"TOKEN_GENERATION_FAILED",
		"Failed to generate token",
		http.StatusInternalServerError,
		"token",
	)
	ErrInvalidActionType = NewAppError(
		"INVALID_ACTION_TYPE",
		"Invalid action type for this operation",
		http.StatusBadRequest,
		"action_type",
	)
	ErrTokenInvalid = NewAppError(
		"INVALID_TOKEN",
		"Invalid token",
		http.StatusUnauthorized,
		"token",
	)
	ErrTokenExpired = NewAppError(
		"TOKEN_EXPIRED",
		"Token has expired",
		http.StatusUnauthorized,
		"token",
	)
	ErrTokenMissing = NewAppError(
		"TOKEN_MISSING",
		"Token is missing",
		http.StatusBadRequest,
		"token",
	)
)

// Session Errors
var (
	ErrSessionNotFound = NewAppError(
		"SESSION_NOT_FOUND",
		"Session not found",
		http.StatusNotFound,
		"session",
	)
	ErrSessionExpired = NewAppError(
		"SESSION_EXPIRED",
		"Session expired",
		http.StatusUnauthorized,
		"session",
	)
	ErrInvalidSession = NewAppError(
		"INVALID_SESSION",
		"Invalid session",
		http.StatusUnauthorized,
		"session",
	)
	ErrSessionCreationFailed = NewAppError(
		"SESSION_CREATION_FAILED",
		"Failed to create session",
		http.StatusInternalServerError,
		"session",
	)
	ErrSessionInvalidationFailed = NewAppError(
		"SESSION_INVALIDATION_FAILED",
		"Failed to invalidate session",
		http.StatusInternalServerError,
		"session",
	)
)

// Auth Method Errors
var (
	ErrAuthMethodNotFound = NewAppError(
		"AUTH_METHOD_NOT_FOUND",
		"Authentication method not found",
		http.StatusNotFound,
		"auth_method",
	)
	ErrAuthMethodUpdateFailed = NewAppError(
		"AUTH_METHOD_UPDATE_FAILED",
		"Failed to update authentication method",
		http.StatusBadRequest,
		"auth_method",
	)
	ErrAuthMethodCreateFailed = NewAppError(
		"AUTH_METHOD_CREATE_FAILED",
		"Failed to create authentication method",
		http.StatusBadRequest,
		"auth_method",
	)
	ErrAuthMethodFindFailed = NewAppError(
		"AUTH_METHOD_FIND_FAILED",
		"Failed to find authentication method",
		http.StatusNotFound,
		"auth_method",
	)
	ErrAuthMethodAlreadyExists = NewAppError(
		"AUTH_METHOD_EXISTS",
		"Authentication method already exists",
		http.StatusConflict,
		"auth_method",
	)
	ErrAuthMethodEnableFailed = NewAppError(
		"AUTH_METHOD_ENABLE_FAILED",
		"Failed to enable authentication method",
		http.StatusBadRequest,
		"auth_method",
	)
	ErrAuthMethodDisableFailed = NewAppError(
		"AUTH_METHOD_DISABLE_FAILED",
		"Failed to disable authentication method",
		http.StatusBadRequest,
		"auth_method",
	)
	ErrAuthMethodValidationFailed = NewAppError(
		"AUTH_METHOD_VALIDATION_FAILED",
		"Authentication method validation failed",
		http.StatusBadRequest,
		"auth_method",
	)
	ErrAuthMethodUnauthorized = NewAppError(
		"AUTH_METHOD_UNAUTHORIZED",
		"Unauthorized to access this authentication method",
		http.StatusUnauthorized,
		"auth_method",
	)
	ErrAuthMethodInvalidType = NewAppError(
		"AUTH_METHOD_INVALID_TYPE",
		"Invalid authentication method type",
		http.StatusBadRequest,
		"auth_method",
	)
	ErrAuthMethodTOTPNotEnabled = NewAppError(
		"TOTP_NOT_ENABLED",
		"TOTP is not enabled for this user",
		http.StatusBadRequest,
		"totp",
	)
	ErrAuthMethodDeleteFailed = NewAppError(
		"AUTH_METHOD_DELETE_FAILED",
		"Failed to delete authentication method",
		http.StatusBadRequest,
		"auth_method",
	)
)

// Premium Key Errors
var (
	ErrPremiumCodeNotFound = NewAppError(
		"PREMIUM_CODE_NOT_FOUND",
		"Premium code not found",
		http.StatusNotFound,
		"premium_code",
	)
	ErrPremiumCodeExpired = NewAppError(
		"PREMIUM_CODE_EXPIRED",
		"Premium code has expired",
		http.StatusBadRequest,
		"premium_code",
	)
	ErrPremiumCodeLimitReached = NewAppError(
		"PREMIUM_CODE_LIMIT_REACHED",
		"Premium code usage limit reached",
		http.StatusBadRequest,
		"premium_code",
	)
	ErrPremiumCodeAlreadyUsed = NewAppError(
		"PREMIUM_CODE_ALREADY_USED",
		"Premium code already used by this user",
		http.StatusBadRequest,
		"premium_code",
	)
	ErrPremiumCodeRedeemFailed = NewAppError(
		"PREMIUM_CODE_REDEEM_FAILED",
		"Failed to redeem premium code",
		http.StatusBadRequest,
		"premium_code",
	)
	ErrPremiumCodeGenerateFailed = NewAppError(
		"PREMIUM_CODE_GENERATE_FAILED",
		"Failed to generate premium code",
		http.StatusBadRequest,
		"premium_code",
	)
	ErrPremiumCodeCountFailed = NewAppError(
		"PREMIUM_CODE_COUNT_FAILED",
		"Failed to count premium codes",
		http.StatusInternalServerError,
		"premium_code",
	)

	// Pending Auth Errors
	ErrServerInternalPendingAuth = NewAppError(
		"SERVER_INTERNAL_PENDING_AUTH",
		"Server internal error for pending auth",
		http.StatusInternalServerError,
		"pending_auth",
	)
	ErrCreatePendingAuthFailed = NewAppError(
		"CREATE_PENDING_AUTH_FAILED",
		"Failed to create pending auth",
		http.StatusBadRequest,
		"pending_auth",
	)
	ErrPendingAuthNotFound = NewAppError(
		"PENDING_AUTH_NOT_FOUND",
		"Pending auth not found",
		http.StatusNotFound,
		"pending_auth",
	)
	ErrPendingAuthExpired = NewAppError(
		"PENDING_AUTH_EXPIRED",
		"Pending auth has expired",
		http.StatusBadRequest,
		"pending_auth",
	)
	ErrPendingAuthInvalid = NewAppError(
		"PENDING_AUTH_INVALID",
		"Invalid pending auth",
		http.StatusBadRequest,
		"pending_auth",
	)
	ErrPendingAuthDeleteFailed = NewAppError(
		"PENDING_AUTH_DELETE_FAILED",
		"Failed to delete pending auth",
		http.StatusBadRequest,
		"pending_auth",
	)
	ErrPendingAuthAttemptsExceeded = NewAppError(
		"PENDING_AUTH_ATTEMPTS_EXCEEDED",
		"Too many attempts. Please Login again.",
		http.StatusBadRequest,
		"pending_auth",
	)

	// User Auth History Errors
	ErrUserHistoryCreateFailed = NewAppError(
		"USER_HISTORY_CREATION_FAILED",
		"Failed to create user history record",
		http.StatusBadRequest,
		"user_history",
	)
	ErrUserHistoryCountFailed = NewAppError(
		"USER_HISTORY_COUNT_FAILED",
		"Failed to count user history",
		http.StatusInternalServerError,
		"user_history",
	)
	ErrUserHistoryFindFailed = NewAppError(
		"USER_HISTORY_FIND_FAILED",
		"Failed to find user history",
		http.StatusBadRequest,
		"user_history",
	)
	ErrUserHistoryUpdateFailed = NewAppError(
		"USER_HISTORY_UPDATE_FAILED",
		"Failed to update user history",
		http.StatusBadRequest,
		"user_history",
	)
	ErrUserHistoryDeleteFailed = NewAppError(
		"USER_HISTORY_DELETE_FAILED",
		"Failed to delete user history",
		http.StatusBadRequest,
		"user_history",
	)
	ErrUserHistoryNotFound = NewAppError(
		"USER_HISTORY_NOT_FOUND",
		"User history not found",
		http.StatusNotFound,
		"user_history",
	)
	ErrInvalidHistoryData = NewAppError(
		"INVALID_HISTORY_DATA",
		"Invalid or corrupted history data",
		http.StatusInternalServerError,
		"history",
	)
	ErrServerInternalHistory = NewAppError(
		"SERVER_INTERNAL_HISTORY",
		"Server internal error for history",
		http.StatusInternalServerError,
		"history",
	)
)
