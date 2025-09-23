# Universal Error Response System

## Overview

The universal error response system provides a standardized way to handle errors across all controllers in the application. This ensures consistent error responses and reduces code duplication.

## Key Components

### 1. Universal Error Handler (`utils/response-handler.go`)

The main error handling function that maps specific errors to standardized HTTP responses:

```go
utils.HandleError(ctx, err, userID)
```

### 2. Custom Error Response Function

For specific error cases not covered by the universal handler:

```go
utils.SendErrorResponse(ctx, statusCode, errorCode, message, field, userID)
```

### 3. Success Response Functions

Standardized success response helpers:

```go
utils.SendCreatedResponse(ctx, data, message)  // 201 Created
utils.SendOKResponse(ctx, data, message)       // 200 OK
utils.SendNoContentResponse(ctx, message)      // 204 No Content
```

## Error Categories Covered

### User Errors
- `ErrUserNotFound` → 404 Not Found
- `ErrUserAlreadyExists` → 409 Conflict
- `ErrUserEmailExists` → 409 Conflict
- `ErrUserInvalidCredentials` → 401 Unauthorized
- `ErrUserAccountLocked` → 403 Forbidden
- `ErrUserUnauthorized` → 403 Forbidden

### API Key Errors
- `ErrAPIKeyNotFound` → 404 Not Found
- `ErrAPIKeyLimitReached` → 403 Forbidden
- `ErrAPIKeyCreateFailed` → 400 Bad Request
- `ErrAPIKeyNameExists` → 409 Conflict
- `ErrAPIKeyExpired` → 403 Forbidden
- `ErrAPIKeyUnauthorized` → 401 Unauthorized

### Short Link Errors
- `ErrShortLinkNotFound` → 404 Not Found
- `ErrShortLinkExpired` → 410 Gone
- `ErrShortLinkUnauthorized` → 403 Forbidden
- `ErrDuplicateShortCode` → 409 Conflict
- `ErrLinkIsBanned` → 403 Forbidden

## Usage Examples

### Before (Manual Error Handling)
```go
if err != nil {
    ctx.JSON(http.StatusNotFound, common.APIResponse{
        Success: false,
        Data:    nil,
        Message: "User not found",
        Error:   map[string]string{"user": "User does not exist"},
    })
    return
}
```

### After (Universal Error Handler)
```go
if err != nil {
    utils.HandleError(ctx, err, userID)
    return
}
```

### Success Response with DTO
```go
// Before
ctx.JSON(http.StatusCreated, common.APIResponse{
    Success: true,
    Data:    gin.H{
        "api_key_id": apiKey.ID,
        "name":       apiKey.Name,
        // ... other fields
    },
    Message: "API key created successfully",
    Error:   nil,
})

// After
response := dto.CreateAPIKeyResponse{
    ID:          apiKey.ID,
    Name:        apiKey.Name,
    CreatedAt:   apiKey.CreatedAt,
    ExpiresAt:   apiKey.ExpiresAt,
    Permissions: apiKey.Permissions,
    Key:         fullAPIKey,
    Warning:     "Please save this key as it will not be shown again.",
}

utils.SendCreatedResponse(ctx, response, "API key created successfully")
```

## Benefits

1. **Consistency**: All error responses follow the same structure
2. **Maintainability**: Error handling logic is centralized
3. **Logging**: Automatic error logging with context
4. **Type Safety**: Using proper DTOs for responses
5. **Code Reduction**: Significantly less boilerplate code

## Migration Guide

To migrate existing controllers:

1. Replace manual `ctx.JSON()` error responses with `utils.HandleError()`
2. Replace success responses with `utils.SendCreatedResponse()`, `utils.SendOKResponse()`, etc.
3. Use proper DTOs for response data instead of `gin.H{}`
4. Remove unused imports (`net/http`, `models/common` if not needed)

## Error Response Format

All error responses follow this standardized format:

```json
{
    "success": false,
    "data": null,
    "message": "Human-readable error message",
    "error": {
        "field_name": "Detailed error description"
    }
}
```

## Success Response Format

All success responses follow this standardized format:

```json
{
    "success": true,
    "data": { /* DTO object or array */ },
    "message": "Success message",
    "error": null
}
```

## Error Codes

Each error type has a standardized error code for API consumers:

- `USER_NOT_FOUND`
- `API_KEY_LIMIT_REACHED`
- `SHORT_LINK_EXPIRED`
- `INVALID_CREDENTIALS`
- etc.

This allows frontend applications to handle specific error types programmatically.