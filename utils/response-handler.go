package utils

import (
	"net/http"

	"github.com/adehusnim37/lihatin-go/models/common"
	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// HandleError provides universal error handling for all controllers
func HandleError(ctx *gin.Context, err error, userID any) {
	Logger.Error("Handling error", "error", err, "user_id", userID)
	switch err {
	// User-related errors
	case ErrUserNotFound:
		SendErrorResponse(ctx, http.StatusNotFound, "USER_NOT_FOUND", "User not found", "user", userID)
	case ErrUserAlreadyExists:
		SendErrorResponse(ctx, http.StatusConflict, "USER_EXISTS", "User already exists", "user", userID)
	case ErrUserEmailExists:
		SendErrorResponse(ctx, http.StatusConflict, "EMAIL_EXISTS", "Email already exists", "email", userID)
	case ErrUserUsernameExists:
		SendErrorResponse(ctx, http.StatusConflict, "USERNAME_EXISTS", "Username already exists", "username", userID)
	case ErrUserCreationFailed:
		SendErrorResponse(ctx, http.StatusBadRequest, "USER_CREATE_FAILED", "Failed to create user", "user", userID)
	case ErrUserUpdateFailed:
		SendErrorResponse(ctx, http.StatusBadRequest, "USER_UPDATE_FAILED", "Failed to update user", "user", userID)
	case ErrUserDeleteFailed:
		SendErrorResponse(ctx, http.StatusBadRequest, "USER_DELETE_FAILED", "Failed to delete user", "user", userID)
	case ErrUserInvalidCredentials:
		SendErrorResponse(ctx, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email/username or password", "auth", userID)
	case ErrUserAccountLocked:
		SendErrorResponse(ctx, http.StatusForbidden, "ACCOUNT_LOCKED", "User account is locked", "user", userID)
	case ErrUserAccountDeleted:
		SendErrorResponse(ctx, http.StatusForbidden, "ACCOUNT_DELETED", "User account has been deleted", "user", userID)
	case ErrUserUnauthorized:
		SendErrorResponse(ctx, http.StatusForbidden, "UNAUTHORIZED", "User is not authorized to perform this action", "auth", userID)
	case ErrUserEmailNotVerified:
		SendErrorResponse(ctx, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "User email is not verified", "email", userID)
	case ErrUserFindFailed:
		SendErrorResponse(ctx, http.StatusBadRequest, "USER_FIND_FAILED", "Failed to find user", "user", userID)
	case ErrUserFailed:
		SendErrorResponse(ctx, http.StatusBadRequest, "USER_FAILED", "Failed to process user request", "user", userID)

	// Email verification-related errors
	case ErrEmailVerificationTokenInvalid:
		SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_VERIFICATION_TOKEN", "Invalid email verification token", "token", userID)
	case ErrEmailVerificationTokenExpired:
		SendErrorResponse(ctx, http.StatusBadRequest, "VERIFICATION_TOKEN_EXPIRED", "Email verification token has expired", "token", userID)
	case ErrEmailAlreadyVerified:
		SendErrorResponse(ctx, http.StatusBadRequest, "EMAIL_ALREADY_VERIFIED", "Email is already verified", "email", userID)
	case ErrEmailVerificationFailed:
		SendErrorResponse(ctx, http.StatusBadRequest, "EMAIL_VERIFICATION_FAILED", "Failed to verify email", "email", userID)
	case ErrCreateVerificationTokenFailed:
		SendErrorResponse(ctx, http.StatusBadRequest, "CREATE_VERIFICATION_TOKEN_FAILED", "Failed to create email verification token", "token", userID)

	// API Key-related errors
	case ErrAPIKeyNotFound:
		SendErrorResponse(ctx, http.StatusNotFound, "API_KEY_NOT_FOUND", "API key not found", "api_key", userID)
	case ErrAPIKeyLimitReached:
		SendErrorResponse(ctx, http.StatusForbidden, "API_KEY_LIMIT_REACHED", "You have reached the maximum number of API keys allowed", "api_key", userID)
	case ErrAPIKeyCreateFailed:
		SendErrorResponse(ctx, http.StatusBadRequest, "API_KEY_CREATE_FAILED", "Failed to create API key", "api_key", userID)
	case ErrAPIKeyFailedFetching:
		SendErrorResponse(ctx, http.StatusNotFound, "API_KEY_FETCH_FAILED", "Could not find API key with provided data", "api_key", userID)
	case ErrAPIKeyNameExists:
		SendErrorResponse(ctx, http.StatusConflict, "API_KEY_NAME_EXISTS", "An API key with this name already exists", "name", userID)
	case ErrAPIKeyRevoked:
		SendErrorResponse(ctx, http.StatusForbidden, "API_KEY_REVOKED", "API key has been revoked", "api_key", userID)
	case ErrAPIKeyExpired:
		SendErrorResponse(ctx, http.StatusForbidden, "API_KEY_EXPIRED", "API key has expired", "api_key", userID)
	case ErrAPIKeyUnauthorized:
		SendErrorResponse(ctx, http.StatusUnauthorized, "API_KEY_UNAUTHORIZED", "Unauthorized: invalid API key", "api_key", userID)
	case ErrAPIKeyInsufficientPerm:
		SendErrorResponse(ctx, http.StatusForbidden, "API_KEY_INSUFFICIENT_PERMISSIONS", "Insufficient permissions for this API key", "permissions", userID)
	case ErrAPIKeyInactive:
		SendErrorResponse(ctx, http.StatusForbidden, "API_KEY_INACTIVE", "API key is inactive", "api_key", userID)
	case ErrAPIKeyInvalidFormat:
		SendErrorResponse(ctx, http.StatusBadRequest, "API_KEY_INVALID_FORMAT", "Invalid API key format", "api_key", userID)
	case ErrAPIKeyMissing:
		SendErrorResponse(ctx, http.StatusBadRequest, "API_KEY_MISSING", "API key is missing", "api_key", userID)
	case ErrAPIKeyInvalidIP:
		SendErrorResponse(ctx, http.StatusForbidden, "API_KEY_INVALID_IP", "API key cannot be used from this IP address", "api_key", userID)
	case ErrAPIKeyInvalidReferrer:
		SendErrorResponse(ctx, http.StatusForbidden, "API_KEY_INVALID_REFERRER", "API key cannot be used from this referrer", "api_key", userID)
	case ErrAPIKeyRateLimitExceeded:
		SendErrorResponse(ctx, http.StatusTooManyRequests, "API_KEY_RATE_LIMIT_EXCEEDED", "API key rate limit exceeded", "api_key", userID)
		
	// Short Link-related errors
	case ErrShortLinkNotFound:
		SendErrorResponse(ctx, http.StatusNotFound, "SHORT_LINK_NOT_FOUND", "Short link not found", "short_link", userID)
	case ErrShortLinkExpired:
		SendErrorResponse(ctx, http.StatusGone, "SHORT_LINK_EXPIRED", "Short link has expired", "short_link", userID)
	case ErrShortLinkInactive:
		SendErrorResponse(ctx, http.StatusForbidden, "SHORT_LINK_INACTIVE", "Short link is inactive", "short_link", userID)
	case ErrShortLinkUnauthorized:
		SendErrorResponse(ctx, http.StatusForbidden, "SHORT_LINK_UNAUTHORIZED", "Unauthorized to access this link", "short_link", userID)
	case ErrDuplicateShortCode:
		SendErrorResponse(ctx, http.StatusConflict, "DUPLICATE_SHORT_CODE", "Short code already exists", "short_code", userID)
	case ErrInvalidOriginalURL:
		SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_ORIGINAL_URL", "Invalid original URL", "original_url", userID)
	case ErrLinkIsBanned:
		SendErrorResponse(ctx, http.StatusForbidden, "LINK_BANNED", "The original URL is banned", "original_url", userID)
	case ErrClickLimitReached:
		SendErrorResponse(ctx, http.StatusForbidden, "CLICK_LIMIT_REACHED", "Click limit reached", "short_link", userID)
	case ErrPasscodeRequired:
		SendErrorResponse(ctx, http.StatusUnauthorized, "PASSCODE_REQUIRED", "Passcode required", "passcode", userID)
	case ErrPasscodeIncorrect:
		SendErrorResponse(ctx, http.StatusUnauthorized, "PASSCODE_INCORRECT", "Incorrect passcode", "passcode", userID)
	case ErrStatsNotEnabled:
		SendErrorResponse(ctx, http.StatusBadRequest, "STATS_NOT_ENABLED", "Statistics tracking is not enabled for this link", "short_link", userID)
	case ErrShortLinkAlreadyDeleted:
		SendErrorResponse(ctx, http.StatusBadRequest, "SHORT_LINK_ALREADY_DELETED", "Short link has already been deleted", "short_link", userID)
	case ErrSomeShortLinksNotFound:
		SendErrorResponse(ctx, http.StatusNotFound, "SOME_SHORT_LINKS_NOT_FOUND", "Some short links not found", "short_links", userID)
	case ErrBulkCreateLimitExceeded:
		SendErrorResponse(ctx, http.StatusBadRequest, "BULK_CREATE_LIMIT_EXCEEDED", "Bulk create limit exceeded, maximum 15 links per batch", "links", userID)
	case ErrBulkCreateFailed:
		SendErrorResponse(ctx, http.StatusBadRequest, "BULK_CREATE_FAILED", "Failed to create bulk short links", "links", userID)
	case ErrEmptyBulkLinksList:
		SendErrorResponse(ctx, http.StatusBadRequest, "EMPTY_BULK_LINKS_LIST", "Bulk links list cannot be empty", "links", userID)

	// Passcode-related errors
	case ErrInvalidPasscode:
		SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_PASSCODE", "Invalid passcode format", "passcode", userID)

	// Click Limit-related errors
	case ErrInvalidClickLimit:
		SendErrorResponse(ctx, http.StatusBadRequest, "INVALID_CLICK_LIMIT", "Click limit must be a positive integer", "click_limit", userID)

	// Code Generation-related errors
	case ErrFailedToGenerateCode:
		SendErrorResponse(ctx, http.StatusInternalServerError, "CODE_GENERATION_FAILED", "Failed to generate unique short code", "short_code", userID)

	// Activity Log-related errors
	case ErrActivityLogNotFound:
		SendErrorResponse(ctx, http.StatusNotFound, "ACTIVITY_LOG_NOT_FOUND", "Activity log not found", "activity_log", userID)
	case ErrActivityLogFailed:
		SendErrorResponse(ctx, http.StatusBadRequest, "ACTIVITY_LOG_FAILED", "Failed to process activity log", "activity_log", userID)

	// Default case for unknown errors
	default:
		Logger.Error("Unhandled error occurred",
			"error", err.Error(),
			"user_id", userID,
		)
		SendErrorResponse(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error", "server", userID)
	}
}

// SendErrorResponse sends a standardized error response
func SendErrorResponse(ctx *gin.Context, statusCode int, errorCode, message, field string, userID interface{}) {
	Logger.Error("API Error",
		"status_code", statusCode,
		"error_code", errorCode,
		"message", message,
		"field", field,
		"user_id", userID,
	)

	ctx.JSON(statusCode, common.APIResponse{
		Success: false,
		Data:    nil,
		Message: message,
		Error:   map[string]string{field: message},
	})
}

// SendSuccessResponse sends a standardized success response
func SendSuccessResponse(ctx *gin.Context, statusCode int, data interface{}, message string) {
	ctx.JSON(statusCode, common.APIResponse{
		Success: true,
		Data:    data,
		Message: message,
		Error:   nil,
	})
}

// SendCreatedResponse sends a standardized 201 Created response
func SendCreatedResponse(ctx *gin.Context, data interface{}, message string) {
	SendSuccessResponse(ctx, http.StatusCreated, data, message)
}

// SendOKResponse sends a standardized 200 OK response
func SendOKResponse(ctx *gin.Context, data interface{}, message string) {
	SendSuccessResponse(ctx, http.StatusOK, data, message)
}

// SendNoContentResponse sends a standardized 204 No Content response
func SendNoContentResponse(ctx *gin.Context, message string) {
	ctx.JSON(http.StatusNoContent, common.APIResponse{
		Success: true,
		Data:    nil,
		Message: message,
		Error:   nil,
	})
}
