package utils

import "errors"

var (
	// Repository errors
	ErrShortLinkNotFound       = errors.New("short link not found")
	ErrShortLinkExpired        = errors.New("short link has expired")
	ErrShortLinkInactive       = errors.New("short link is inactive")
	ErrShortLinkUnauthorized   = errors.New("unauthorized to access this link")
	ErrDuplicateShortCode      = errors.New("short code already exists")
	ErrInvalidOriginalURL      = errors.New("invalid original URL")
	ErrShortLinkAlreadyDeleted = errors.New("short link has already been deleted")
	ErrEmptyCodesList          = errors.New("codes list cannot be empty")
	ErrSomeShortLinksNotFound  = errors.New("some short links not found")
	ErrLinkIsBanned            = errors.New("the original URL is banned")
	ErrShortIsNotDeleted       = errors.New("short link is not deleted")

	// Service errors
	ErrFailedToGenerateCode = errors.New("failed to generate unique short code")
	ErrInvalidPasscode      = errors.New("invalid passcode format")
	ErrInvalidClickLimit    = errors.New("click limit must be a positive integer")
	ErrStatsNotEnabled      = errors.New("statistics tracking is not enabled for this link")

	// Passcode errors
	ErrPasscodeRequired  = errors.New("passcode required")
	ErrPasscodeIncorrect = errors.New("incorrect passcode")

	// Limit errors
	ErrClickLimitReached = errors.New("click limit reached")
)

var (
	// API key errors
	ErrAPIKeyFailedFetching    = errors.New("failed to fetch API key")
	ErrAPIKeyNotFound          = errors.New("API key not found")
	ErrAPIKeyRevoked           = errors.New("API key has been revoked")
	ErrAPIKeyUnauthorized      = errors.New("unauthorized to access this API key")
	ErrAPIKeyNameExists        = errors.New("API key with the same name already exists")
	ErrAPIKeyInvalidFormat     = errors.New("invalid API key format")
	ErrAPIKeyInsufficientPerm  = errors.New("insufficient permissions for this API key")
	ErrAPIKeyMissing           = errors.New("API key is missing")
	ErrAPIKeyInactive          = errors.New("API key is inactive")
	ErrAPIKeyInvalidIP         = errors.New("API key cannot be used from this IP address")
	ErrAPIKeyInvalidReferrer   = errors.New("API key cannot be used from this referrer")
	ErrAPIKeyExpired           = errors.New("API key has expired")
	ErrAPIKeyCreateFailed      = errors.New("failed to create API key")
	ErrAPIKeyUpdateFailed      = errors.New("failed to update API key")
	ErrAPIKeyIDFailedFormat    = errors.New("invalid API key ID format")
	ErrAPIKeyLimitReached      = errors.New("API key limit reached for user")
	ErrAPIKeyRateLimitExceeded = errors.New("API key rate limit exceeded")
	ErrAPIKeyIPNotAllowed      = errors.New("API key cannot be used from this IP address")
	ErrGenerateAPIKeyFailed	 = errors.New("failed to generate API key")
)

var (
	// User errors
	ErrUserNotFound           = errors.New("user not found")
	ErrUserAlreadyExists      = errors.New("user already exists")
	ErrUserEmailExists        = errors.New("email already exists")
	ErrUserUsernameExists     = errors.New("username already exists")
	ErrUserCreationFailed     = errors.New("failed to create user")
	ErrUserUpdateFailed       = errors.New("failed to update user")
	ErrUserDeleteFailed       = errors.New("failed to delete user")
	ErrUserLockFailed         = errors.New("failed to lock user")
	ErrUserUnlockFailed       = errors.New("failed to unlock user")
	ErrUserAlreadyLocked      = errors.New("user is already locked")
	ErrUserNotLocked          = errors.New("user is not locked")
	ErrUserInvalidCredentials = errors.New("invalid email/username or password")
	ErrUserAccountLocked      = errors.New("user account is locked")
	ErrUserAccountDeleted     = errors.New("user account has been deleted")
	ErrUserPasswordHashFailed = errors.New("failed to hash user password")
	ErrUserInvalidInput       = errors.New("invalid user input data")
	ErrUserUnauthorized       = errors.New("user is not authorized to perform this action")
	ErrUserValidationFailed   = errors.New("user data validation failed")
	ErrUserDuplicateEntry     = errors.New("duplicate entry for user data")
	ErrUserDatabaseError      = errors.New("database error while processing user request")
	ErrUserEmailNotVerified   = errors.New("user email is not verified")
)

var (
	// Activity log errors
	ErrActivityLogNotFound = errors.New("activity log not found")
	ErrActivityLogFailed   = errors.New("failed to process activity log")
)

var (
	// Email verification errors
	ErrEmailVerificationFailed       = errors.New("failed to verify email")
	ErrEmailVerificationTokenInvalid = errors.New("invalid email verification token")
	ErrEmailVerificationTokenExpired = errors.New("email verification token has expired")
	ErrEmailAlreadyVerified          = errors.New("email is already verified")
	ErrCreateVerificationTokenFailed = errors.New("failed to create email verification token")
)

var (
	// Token generation errors
	ErrTokenGenerationFailed = errors.New("failed to generate token")
	ErrTokenInvalid          = errors.New("invalid token")
	ErrTokenExpired          = errors.New("token has expired")
)
