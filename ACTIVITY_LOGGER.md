# Enhanced Activity Logger Documentation

## Overview
The enhanced activity logger middleware captures comprehensive request/response data including request bodies, query parameters, route parameters, context locals, and response times. This provides detailed audit trails for all API interactions.

## Features Added

### 1. Request Body Logging
- **Captures**: Full request body for POST, PUT, PATCH requests
- **Security**: Automatically redacts sensitive fields (password, token, secret, key, auth, authorization)
- **Size Limit**: Truncates bodies larger than 1000 characters to prevent log bloat
- **Format**: JSON structure preserved when possible

### 2. Query Parameters Logging
- **Captures**: All URL query parameters
- **Format**: JSON object with parameter names and values
- **Example**: `{"page": "1", "limit": "10", "search": "john"}`

### 3. Route Parameters Logging
- **Captures**: All route/path parameters (e.g., `:id`, `:username`)
- **Format**: JSON object mapping parameter names to values
- **Example**: `{"id": "123", "username": "johndoe"}`

### 4. Context Locals Logging
- **Captures**: Important context values set during request processing
- **Included Fields**:
  - `user_id` - Authenticated user identifier
  - `session_id` - Session identifier
  - `request_id` - Request tracking ID
  - `tenant_id` - Multi-tenant identifier
  - `role` - User role/permission level
  - `permissions` - User permissions
  - `x_request_id` - X-Request-ID header
  - `x_forwarded_for` - X-Forwarded-For header

### 5. Response Time Logging
- **Captures**: Request processing time in milliseconds
- **Usage**: Performance monitoring and optimization

## Database Schema Updates

You need to update your `ActivityLog` table to include the new fields:

```sql
ALTER TABLE ActivityLog 
ADD COLUMN requestbody TEXT,
ADD COLUMN queryparams TEXT,
ADD COLUMN routeparams TEXT,
ADD COLUMN contextlocals TEXT,
ADD COLUMN responsetime BIGINT;
```

### Complete Updated Schema
```sql
CREATE TABLE ActivityLog (
    id VARCHAR(36) PRIMARY KEY,
    level VARCHAR(50) NOT NULL,
    message VARCHAR(255) NOT NULL,
    username VARCHAR(50) NOT NULL,
    timestamp VARCHAR(50) NOT NULL,
    ipaddress VARCHAR(50) NOT NULL,
    useragent VARCHAR(255) NOT NULL,
    browserinfo VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    route VARCHAR(100) NOT NULL,
    method VARCHAR(10) NOT NULL,
    statuscode INT NOT NULL,
    requestbody TEXT,              -- NEW: Request body content
    queryparams TEXT,              -- NEW: Query parameters as JSON
    routeparams TEXT,              -- NEW: Route parameters as JSON
    contextlocals TEXT,            -- NEW: Context values as JSON
    responsetime BIGINT,           -- NEW: Response time in milliseconds
    createdat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updatedat TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deletedat TIMESTAMP NULL
);
```

## Enhanced LoggerUser Model

The model now includes these additional fields:

```go
type LoggerUser struct {
    // ... existing fields ...
    RequestBody   string `json:"request_body,omitempty"`     // Request body (sanitized)
    QueryParams   string `json:"query_params,omitempty"`     // Query parameters as JSON
    RouteParams   string `json:"route_params,omitempty"`     // Route parameters as JSON
    ContextLocals string `json:"context_locals,omitempty"`   // Context values as JSON
    ResponseTime  int64  `json:"response_time,omitempty"`    // Response time in milliseconds
    // ... existing fields ...
}
```

## API Response Examples

### Enhanced Log Entry Example
```json
{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "level": "INFO",
    "message": "POST /v1/users - 201 (45ms)",
    "username": "johndoe",
    "timestamp": "2025-05-26T10:30:00Z",
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0...",
    "browser_info": "Mozilla/5.0...",
    "action": "Create",
    "route": "/v1/users",
    "method": "POST",
    "status_code": 201,
    "request_body": "{\"username\":\"newuser\",\"email\":\"user@example.com\",\"password\":\"[REDACTED]\"}",
    "query_params": "{\"source\":\"web\",\"ref\":\"signup\"}",
    "route_params": "{}",
    "context_locals": "{\"user_id\":\"123\",\"session_id\":\"sess_456\",\"role\":\"admin\"}",
    "response_time": 45,
    "created_at": "2025-05-26T10:30:00Z",
    "updated_at": "2025-05-26T10:30:00Z"
}
```

## Security Features

### 1. Sensitive Data Redaction
The following fields are automatically redacted in request bodies:
- `password`
- `token`
- `secret` 
- `key`
- `auth`
- `authorization`

### 2. Size Limitations
- Request bodies are truncated at 1000 characters
- Large JSON objects are preserved structurally but content may be truncated

### 3. Async Logging
- Logs are saved asynchronously to prevent blocking API responses
- Failed log saves are logged to console but don't affect API functionality

## Usage Examples

### Setting Context Values for Logging
```go
// In your authentication middleware
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... authentication logic ...
        
        // Set values that will be captured in logs
        c.Set("user_id", user.ID)
        c.Set("username", user.Username)
        c.Set("role", user.Role)
        c.Set("session_id", sessionID)
        
        c.Next()
    }
}
```

### Custom Request ID Middleware
```go
func RequestIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        
        c.Set("request_id", requestID)
        c.Header("X-Request-ID", requestID)
        
        c.Next()
    }
}
```

## Performance Considerations

### 1. Memory Usage
- Request bodies are read into memory before being restored
- Large request bodies may impact memory usage
- Consider setting appropriate request size limits

### 2. Database Storage
- TEXT fields can store large amounts of data
- Consider implementing log rotation/archival strategies
- Monitor database growth for high-traffic applications

### 3. Async Processing
- Logging operations are non-blocking
- Failed logs don't impact API response times
- Consider implementing retry mechanisms for critical logging

## Monitoring and Analytics

### Useful Queries

#### Performance Analysis
```sql
-- Average response times by endpoint
SELECT route, method, AVG(responsetime) as avg_response_time 
FROM ActivityLog 
WHERE responsetime IS NOT NULL 
GROUP BY route, method 
ORDER BY avg_response_time DESC;
```

#### Error Rate Analysis
```sql
-- Error rates by endpoint
SELECT route, method, 
       COUNT(*) as total_requests,
       SUM(CASE WHEN statuscode >= 400 THEN 1 ELSE 0 END) as error_count,
       (SUM(CASE WHEN statuscode >= 400 THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) as error_rate
FROM ActivityLog 
GROUP BY route, method
ORDER BY error_rate DESC;
```

#### User Activity Analysis
```sql
-- Most active users
SELECT username, COUNT(*) as request_count
FROM ActivityLog 
WHERE createdat >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
GROUP BY username 
ORDER BY request_count DESC 
LIMIT 10;
```

#### Request Body Analysis
```sql
-- Find requests with specific patterns in body
SELECT * FROM ActivityLog 
WHERE requestbody LIKE '%email%' 
  AND action = 'Create' 
  AND route = '/v1/users'
ORDER BY createdat DESC;
```

## Best Practices

### 1. Context Usage
- Set meaningful context values in middlewares
- Use consistent naming conventions for context keys
- Avoid storing sensitive data in context

### 2. Log Management
- Implement log rotation policies
- Set up monitoring for log storage growth
- Consider separating audit logs from application logs

### 3. Security
- Regularly review redacted field patterns
- Monitor for potential PII leakage
- Implement access controls for log viewing

### 4. Performance
- Monitor response time impacts
- Consider selective logging for high-frequency endpoints
- Implement sampling for very high-traffic scenarios

## Troubleshooting

### Common Issues

#### 1. Missing Request Body
- Ensure middleware is placed before other middlewares that consume the body
- Check that request Content-Type is properly set

#### 2. Empty Context Locals
- Verify that context values are set before the activity logger runs
- Check middleware ordering

#### 3. Database Errors
- Ensure new columns are added to the database
- Check column size limits for large request bodies

#### 4. Performance Issues
- Monitor database query performance
- Consider indexing frequently queried fields
- Implement log archival strategies

## Migration Guide

### Updating Existing Installation

1. **Update Database Schema**:
   ```sql
   ALTER TABLE ActivityLog 
   ADD COLUMN requestbody TEXT,
   ADD COLUMN queryparams TEXT,
   ADD COLUMN routeparams TEXT,
   ADD COLUMN contextlocals TEXT,
   ADD COLUMN responsetime BIGINT;
   ```

2. **Update Code**: The middleware and model changes are backward compatible

3. **Monitor**: Watch for any performance impacts after deployment

4. **Verify**: Test that new fields are being populated correctly

### Rollback Procedure
If issues arise, you can temporarily disable new field logging by modifying the middleware to not set these fields, while keeping the database schema for future use.
