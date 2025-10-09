# JWT Blacklist & Refresh Token System

## Overview

This document describes the complete JWT blacklist and refresh token implementation that ensures proper token revocation during logout and token refresh functionality.

## System Architecture

### 1. JWT with JTI (JWT ID)
- Each JWT now includes a unique `jti` (JWT ID) in the `RegisteredClaims.ID` field
- JTI is a 64-character hex string (32 random bytes)
- Used for tracking and blacklisting individual tokens

### 2. Refresh Token System
- Separate long-lived tokens stored in Redis (7 days)
- Used to obtain new access tokens without re-authentication
- Stored with metadata: UserID, SessionID, DeviceID, LastIP, timestamps
- Automatically rotated on each refresh for security

### 3. JWT Blacklist
- Tracks revoked JWT tokens in Redis
- Prevents usage of valid-but-logged-out tokens
- TTL matches JWT expiration (auto-cleanup)

## Flow Diagrams

### Login Flow
```
┌─────────────┐
│  User Login │
└──────┬──────┘
       │
       ▼
┌──────────────────┐
│ Create Session   │ ───► Redis: session:{id}
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Generate JWT     │ ───► JWT with jti
│ (with jti)       │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Generate Refresh │ ───► Redis: refresh_token:{token}
│ Token            │      {UserID, SessionID, DeviceID, LastIP}
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Return Both      │ ───► {token, refresh_token}
│ Tokens           │
└──────────────────┘
```

### Refresh Token Flow
```
┌──────────────────┐
│ Client sends     │
│ refresh_token    │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Validate from    │ ◄─── Redis: refresh_token:{token}
│ Redis            │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Get User Data    │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Generate new JWT │ ───► New JWT with new jti
│ (same session)   │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Delete old       │ ───► Redis: DEL old token
│ refresh token    │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Generate new     │ ───► Redis: new refresh token
│ refresh token    │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Return both new  │ ───► {token, refresh_token}
│ tokens           │
└──────────────────┘
```

### Logout Flow
```
┌──────────────────┐
│ User Logout      │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Extract JWT      │
│ claims (jti)     │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Blacklist JWT    │ ───► Redis: jwt_blacklist:{jti} = 1
│ (add jti)        │      TTL = JWT expiration
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Delete Refresh   │ ───► Redis: DEL all user refresh tokens
│ Tokens           │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Delete Session   │ ───► Redis: DEL session:{id}
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Return Success   │
└──────────────────┘
```

### Middleware Validation Flow
```
┌──────────────────┐
│ Incoming Request │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Extract JWT      │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Validate JWT     │
│ signature & exp  │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Check Blacklist  │ ◄─── Redis: EXISTS jwt_blacklist:{jti}
└──────┬───────────┘
       │
       ├─ Blacklisted ──► 401 Unauthorized
       │
       ▼
┌──────────────────┐
│ Verify User      │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Validate Session │ ◄─── Redis: session:{id}
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Allow Request    │
└──────────────────┘
```

## Redis Data Structures

### 1. JWT Blacklist Entry
```
Key:   jwt_blacklist:{jti}
Type:  String
Value: "1"
TTL:   Matches JWT expiration time

Example:
jwt_blacklist:a1b2c3d4e5f6... = "1"
TTL: 24 hours (or whatever JWT_EXPIRED is set to)
```

### 2. Refresh Token Entry
```
Key:   refresh_token:{token}
Type:  Hash
Fields:
  - user_id: "user-uuid"
  - session_id: "sess_abc123..."
  - device_id: "device-fingerprint"
  - last_ip: "192.168.1.1"
  - created_at: unix_timestamp
  - expires_at: unix_timestamp
TTL:   7 days (604800 seconds)

Example:
refresh_token:e7f8g9h0i1j2... {
  user_id: "550e8400-e29b-41d4-a716-446655440000"
  session_id: "sess_abc123def456..."
  device_id: "Chrome-Windows-10"
  last_ip: "192.168.1.100"
  created_at: 1704067200
  expires_at: 1704672000
}
```

### 3. Session Entry (Existing)
```
Key:   session:{session_id}
Type:  JSON String
Value: Complete session object
TTL:   48 hours (or SESSION_TTL)
```

## Implementation Details

### Files Modified/Created

#### 1. `utils/jwt.go`
- **Modified**: `GenerateJWT()` - Now generates unique `jti` for each token
- **Modified**: `GenerateRefreshToken()` - Creates refresh token and stores in Redis

#### 2. `utils/jwt_blacklist.go` (New)
- **Type**: `JWTBlacklistManager` - Manages JWT blacklist
  - `BlacklistJWT()` - Adds JWT ID to blacklist
  - `IsJWTBlacklisted()` - Checks if JWT is blacklisted
- **Type**: `RefreshTokenManager` - Manages refresh tokens
  - `StoreRefreshToken()` - Stores refresh token in Redis
  - `GetRefreshToken()` - Retrieves refresh token from Redis
  - `DeleteRefreshToken()` - Removes specific refresh token
  - `DeleteAllUserRefreshTokens()` - Removes all user's refresh tokens

#### 3. `utils/tokens.go`
- **Added**: `GenerateSecureToken()` - Generic secure token generator

#### 4. `middleware/auth.go`
- **Modified**: Added blacklist check after JWT validation
- Rejects requests if JWT is blacklisted

#### 5. `controllers/auth/login.go`
- **Modified**: Now generates and returns refresh token along with JWT

#### 6. `controllers/auth/logout.go`
- **Modified**: 
  - Blacklists current JWT
  - Deletes all user refresh tokens
  - Deletes session from Redis

#### 7. `controllers/auth/base.go`
- **Modified**: `RefreshToken()` - Complete rewrite to use Redis-based refresh tokens

#### 8. `utils/session/manager.go`
- **Added**: `GetRedisClient()` - Exposes Redis client for other utilities

#### 9. `utils/session/redis_store.go`
- **Added**: `GetClient()` - Returns Redis client

## API Endpoints

### 1. Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "user@example.com",
  "password": "password123"
}

Response 200:
{
  "success": true,
  "data": {
    "user": {...},
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "a1b2c3d4e5f6...",
    "requires_2fa": false,
    "is_verified": true
  },
  "message": "Login successful"
}
```

### 2. Refresh Token
```http
POST /api/v1/auth/refresh-token
Content-Type: application/json

{
  "refresh_token": "a1b2c3d4e5f6..."
}

Response 200:
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "g7h8i9j0k1l2..."
  },
  "message": "Token refreshed successfully"
}

Response 401 (Invalid/Expired):
{
  "success": false,
  "message": "Invalid or expired refresh token",
  "error": {
    "refresh_token": "Please login again"
  }
}
```

### 3. Logout
```http
POST /api/v1/auth/logout
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...

Response 200:
{
  "success": true,
  "message": "Logout successful"
}
```

## Security Features

### 1. JWT Blacklist
- **Purpose**: Prevents revoked tokens from being used
- **Implementation**: Redis with TTL matching token expiration
- **Benefit**: Immediate token revocation without waiting for expiration

### 2. Refresh Token Rotation
- **Purpose**: Limits damage if refresh token is stolen
- **Implementation**: Old refresh token deleted, new one created on each refresh
- **Benefit**: Stolen tokens become invalid after single use

### 3. Device & IP Tracking
- **Purpose**: Detect suspicious token usage
- **Implementation**: Stored in refresh token metadata
- **Benefit**: Can log warnings when device/IP changes

### 4. Automatic Cleanup
- **Purpose**: Prevent Redis memory bloat
- **Implementation**: TTL on all Redis keys
- **Benefit**: Blacklist and refresh tokens auto-expire

## Environment Variables

```env
# JWT Configuration
JWT_SECRET=your-secret-key-here
JWT_EXPIRED=24  # Access token lifetime in hours

# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Session Configuration
SESSION_TTL=48  # Session lifetime in hours
```

## Token Lifetimes

| Token Type | Default Lifetime | Configurable |
|------------|-----------------|--------------|
| JWT Access Token | 24 hours | Yes (`JWT_EXPIRED`) |
| Refresh Token | 7 days | No (hardcoded) |
| Session | 48 hours | Yes (`SESSION_TTL`) |
| JWT Blacklist | Matches JWT expiration | Automatic |

## Testing

### Test Scenarios

#### 1. Login & Access
```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test@example.com","password":"password123"}'

# Use access token
curl http://localhost:8080/api/v1/auth/profile \
  -H "Authorization: Bearer <access_token>"
```

#### 2. Refresh Token
```bash
# Use refresh token to get new access token
curl -X POST http://localhost:8080/api/v1/auth/refresh-token \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token>"}'
```

#### 3. Logout & Blacklist
```bash
# Logout (blacklists JWT)
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <access_token>"

# Try to use same token (should fail)
curl http://localhost:8080/api/v1/auth/profile \
  -H "Authorization: Bearer <access_token>"
# Expected: 401 Unauthorized - "Token has been revoked"
```

#### 4. Verify Redis Data
```bash
# Check Redis for blacklist
redis-cli
> KEYS jwt_blacklist:*
> GET jwt_blacklist:{jti}

# Check refresh tokens
> KEYS refresh_token:*
> HGETALL refresh_token:{token}

# Check sessions
> KEYS session:*
> GET session:{session_id}
```

## Error Handling

### Common Errors

1. **"Token has been revoked"**
   - JWT is in blacklist (user logged out)
   - Solution: Login again

2. **"Invalid or expired refresh token"**
   - Refresh token not found in Redis
   - Refresh token expired (>7 days)
   - Solution: Login again

3. **"Invalid or expired token"**
   - JWT signature invalid
   - JWT expired
   - Solution: Use refresh token or login

4. **"Failed to check JWT blacklist"**
   - Redis connection error
   - Solution: Check Redis connection

## Monitoring & Logging

### Log Events

1. **JWT Blacklisted**
```
[INFO] JWT blacklisted successfully
  user_id: 550e8400-e29b-41d4-a716-446655440000
  jti: a1b2c3d4... (preview)
```

2. **Token Refreshed**
```
[INFO] Token refreshed successfully
  user_id: 550e8400-e29b-41d4-a716-446655440000
  session_id: sess_abc1... (preview)
```

3. **Blacklist Check Failed**
```
[ERROR] Failed to check JWT blacklist
  error: connection refused
```

4. **Device Mismatch**
```
[WARN] Device mismatch during token refresh
  user_id: 550e8400-e29b-41d4-a716-446655440000
  expected_device: Chrome-Windows-10
  actual_device: Firefox-Ubuntu-20.04
```

### Redis Monitoring

```bash
# Monitor Redis operations
redis-cli MONITOR | grep -E "jwt_blacklist|refresh_token"

# Check memory usage
redis-cli INFO memory

# Count keys by type
redis-cli --scan --pattern "jwt_blacklist:*" | wc -l
redis-cli --scan --pattern "refresh_token:*" | wc -l
```

## Best Practices

1. **Always use HTTPS in production** - Prevent token interception
2. **Keep JWT_SECRET secure** - Use strong random string
3. **Monitor failed refresh attempts** - Potential security issue
4. **Set appropriate token lifetimes** - Balance security and UX
5. **Implement rate limiting** - Prevent brute force on refresh endpoint
6. **Log suspicious activity** - Device/IP changes, multiple refreshes
7. **Regular security audits** - Review token usage patterns

## Migration from Old System

### Old System
- No JWT blacklist
- Refresh tokens were JWTs (not Redis-based)
- Logout only deleted session
- Revoked tokens still worked until expiration

### New System
- JWT blacklist in Redis
- Refresh tokens stored in Redis with metadata
- Logout blacklists JWT + deletes refresh tokens + deletes session
- Revoked tokens rejected immediately

### Breaking Changes
- `utils.GenerateRefreshToken()` signature changed
  - Old: `GenerateRefreshToken(userID string)`
  - New: `GenerateRefreshToken(ctx, redisClient, userID, sessionID, deviceID, lastIP)`
- `utils.ValidateRefreshToken()` no longer needed (Redis-based validation)
- Refresh token response includes new refresh token (token rotation)

## Troubleshooting

### Issue: "Token has been revoked" but user didn't logout
**Cause**: Another device/session logged out, blacklisting the token
**Solution**: Login again

### Issue: Refresh token fails immediately after login
**Cause**: Redis connection issue or token not stored
**Solution**: Check Redis connection and logs

### Issue: Memory usage growing in Redis
**Cause**: Blacklist entries not expiring
**Solution**: Check TTL is set correctly on blacklist keys

### Issue: User can still access after logout
**Cause**: Middleware not checking blacklist
**Solution**: Verify `AuthMiddleware` has blacklist check

## Future Enhancements

1. **Refresh Token Families** - Track refresh token lineage
2. **Device Management UI** - Let users view/revoke active sessions
3. **Geolocation Tracking** - Enhanced security with IP geolocation
4. **Rate Limiting** - Limit refresh token usage per time period
5. **Audit Trail** - Complete history of token usage
6. **Sliding Sessions** - Auto-extend session on activity
7. **Token Introspection Endpoint** - Check token status

## References

- [RFC 7519 - JSON Web Token (JWT)](https://tools.ietf.org/html/rfc7519)
- [OWASP JWT Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html)
- [Redis Best Practices](https://redis.io/docs/manual/patterns/)

---

**Document Version**: 1.0  
**Last Updated**: 2024  
**Author**: System Generated
