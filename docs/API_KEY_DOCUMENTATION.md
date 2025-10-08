# API Key Authentication System

## Overview

This application now supports API key authentication for service-to-service communication. The API key system provides secure programmatic access to the URL shortener functionality without requiring user session authentication.

## Features

- **Secure API Key Generation**: Keys are generated with secure hashing using SHA256
- **Database Validation**: API keys are validated against the database with expiration checking
- **User Association**: Each API key is associated with a user account
- **Permission System**: API keys support custom permissions (JSON field for future extensibility)
- **Rate Limiting**: API routes have higher rate limits (1000 req/min vs 3 req/min for public routes)
- **Comprehensive Logging**: All API key authentication attempts are logged
- **Soft Deletes**: API keys can be safely deleted without losing audit trails

## API Routes

All API key authenticated routes are available under the `/v1/api/short` prefix:

### Available Endpoints

- `POST /v1/api/short` - Create a new short link
- `GET /v1/api/short/:code` - Get details of a specific short link
- `PUT /v1/api/short/:code` - Update a short link
- `GET /v1/api/short` - List all short links for the authenticated user
- `GET /v1/api/short/:code/stats` - Get statistics for a short link
- `GET /v1/api/short/:code/views` - Get paginated view history
- `DELETE /v1/api/short/:code` - Delete a short link
- `GET /v1/api/short/stats` - Get comprehensive statistics

### Authentication

Include your API key in the request header:

```
X-API-Key: lihatin_live_1234567890abcdef...
```

### Example Usage

```bash
# Create a short link via API
curl -X POST "https://your-domain.com/v1/api/short" \
  -H "X-API-Key: lihatin_live_1234567890abcdef..." \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://example.com/very-long-url",
    "custom_code": "mycode"
  }'

# Get short link details
curl -X GET "https://your-domain.com/v1/api/short/mycode" \
  -H "X-API-Key: lihatin_live_1234567890abcdef..."

# List all short links
curl -X GET "https://your-domain.com/v1/api/short" \
  -H "X-API-Key: lihatin_live_1234567890abcdef..."
```

## API Key Management

### API Key Format

API keys follow the format: `lihatin_{environment}_{32_character_key_id}`

- **Test Environment**: `lihatin_test_...`
- **Live Environment**: `lihatin_live_...`

### Security Features

1. **Hashed Storage**: Only SHA256 hashes are stored in the database
2. **Expiration Support**: API keys can have optional expiration dates
3. **Permission Control**: Each key can have specific permissions (JSON field)
4. **Activity Logging**: All API key usage is logged for security auditing
5. **Key Preview**: Only first 8 characters are shown in logs for security

### Database Schema

```sql
-- API Key model fields
id               varchar(191) PRIMARY KEY
user_id          varchar(191) NOT NULL
name             varchar(100) -- Human-readable name for the key
key_hash         varchar(255) -- SHA256 hash of the actual key
permissions      json         -- Custom permissions (future extensibility)
expires_at       datetime     -- Optional expiration
last_used_at     datetime     -- Track usage
created_at       datetime
updated_at       datetime
deleted_at       datetime     -- Soft deletes
```

## Middleware Implementation

The API key authentication is implemented in two middleware functions:

### 1. APIKeyMiddleware

Direct API key repository usage:

```go
middleware.APIKeyMiddleware(apiKeyRepo)
```

### 2. AuthRepositoryAPIKeyMiddleware (Recommended)

Uses the complete auth repository:

```go
middleware.AuthRepositoryAPIKeyMiddleware(authRepo)
```

## Error Responses

### Missing API Key

```json
{
  "success": false,
  "data": null,
  "message": "API key required",
  "error": {
    "api_key": "Missing X-API-Key header"
  }
}
```

### Invalid API Key

```json
{
  "success": false,
  "data": null,
  "message": "Invalid API key",
  "error": {
    "api_key": "The provided API key is invalid or expired"
  }
}
```

## Context Variables

When API key authentication succeeds, the following variables are available in the Gin context:

- `user_id` - The ID of the user associated with the API key
- `username` - Username of the associated user
- `email` - Email of the associated user
- `role` - Role of the associated user
- `is_premium` - Premium status of the user
- `user` - Complete user object
- `api_key` - Complete API key record
- `api_key_id` - ID of the API key
- `api_key_name` - Human-readable name of the API key
- `api_key_permissions` - Permissions JSON for the API key
- `is_api_authenticated` - Boolean flag indicating API authentication

## Rate Limiting

- **Public Routes**: 3 requests per minute
- **User-Authenticated Routes**: 100 requests per minute
- **API Key Routes**: 1000 requests per minute

## Security Considerations

1. **Key Storage**: Never log or display complete API keys
2. **HTTPS Only**: Always use HTTPS in production
3. **Key Rotation**: Implement regular API key rotation
4. **Monitoring**: Monitor API key usage for suspicious activity
5. **Permissions**: Use the permissions field to restrict API key capabilities
6. **Expiration**: Set reasonable expiration dates for API keys

## Future Enhancements

1. **Scoped Permissions**: Implement granular permissions system
2. **Key Rotation**: Automatic key rotation capabilities
3. **Usage Analytics**: Detailed API key usage analytics
4. **Webhook Support**: API keys for webhook authentication
5. **Team Keys**: Shared API keys for team accounts

## Integration Example

```go
// In your application setup
authRepo := repositories.NewAuthRepository(gormDB)

// Apply API key middleware to routes
apiGroup := router.Group("/api")
apiGroup.Use(middleware.AuthRepositoryAPIKeyMiddleware(authRepo))
apiGroup.POST("/short", shortController.Create)
```

## Troubleshooting

### Common Issues

1. **"API key middleware not properly initialized"**
   - Ensure you're using `AuthRepositoryAPIKeyMiddleware` with a valid AuthRepository
   - Avoid using the deprecated `ApiKeyAuthMiddleware()` without parameters

2. **"Missing X-API-Key header"**
   - Ensure the header name is exactly `X-API-Key` (case-sensitive)
   - Check that your HTTP client is sending the header correctly

3. **"Invalid API key"**
   - Verify the API key format matches the expected pattern
   - Check if the API key has expired
   - Ensure the API key exists in the database and is not deleted

### Debugging

Enable debug logging to see API key validation attempts:

```go
utils.Logger.SetLevel(logrus.DebugLevel)
```

This will show logs like:
```
INFO API key authentication successful user_id=123 api_key_id=456 api_key_name="My API Key"
WARN API key validation failed key_preview="lihatin_" error="API key not found"
```