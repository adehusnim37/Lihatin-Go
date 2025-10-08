# Quick Test Plan - JWT Blacklist System

## Test 1: Login and Get Tokens
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test@example.com",
    "password": "password123"
  }'
```

**Expected:**
```json
{
  "success": true,
  "data": {
    "token": "eyJ...",
    "refresh_token": "abc123..."
  }
}
```

**Redis Check:**
```bash
redis-cli
> KEYS session:*
> KEYS refresh_token:*
> HGETALL refresh_token:abc123...
```

---

## Test 2: Use Access Token
```bash
# Save token from Test 1
TOKEN="eyJ..."

curl http://localhost:8080/api/v1/auth/profile \
  -H "Authorization: Bearer $TOKEN"
```

**Expected:**
- 200 OK
- Profile data returned

---

## Test 3: Logout (Blacklist Token)
```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer $TOKEN"
```

**Expected:**
```json
{
  "success": true,
  "message": "Logout successful"
}
```

**Redis Check:**
```bash
redis-cli
> KEYS jwt_blacklist:*
1) "jwt_blacklist:a1b2c3d4..."

> GET jwt_blacklist:a1b2c3d4...
"1"

> KEYS refresh_token:*
(empty array) # All deleted

> KEYS session:*
(empty array) # Deleted
```

---

## Test 4: Try Using Blacklisted Token (Should Fail)
```bash
curl http://localhost:8080/api/v1/auth/profile \
  -H "Authorization: Bearer $TOKEN"
```

**Expected:**
```json
{
  "success": false,
  "message": "Token has been revoked",
  "error": {
    "auth": "Please login again"
  }
}
```

**Status Code:** 401 Unauthorized

---

## Test 5: Refresh Token Flow
```bash
# Login again
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test@example.com",
    "password": "password123"
  }'

# Save tokens
TOKEN="eyJ..."
REFRESH_TOKEN="abc123..."

# Use refresh token
curl -X POST http://localhost:8080/api/v1/auth/refresh-token \
  -H "Content-Type: application/json" \
  -d "{
    \"refresh_token\": \"$REFRESH_TOKEN\"
  }"
```

**Expected:**
```json
{
  "success": true,
  "data": {
    "token": "eyJ... (NEW TOKEN)",
    "refresh_token": "xyz789... (NEW REFRESH TOKEN)"
  }
}
```

**Redis Check:**
```bash
redis-cli
> KEYS refresh_token:*
1) "refresh_token:xyz789..."  # New token
# Old token (abc123...) should be deleted
```

---

## Test 6: Old Refresh Token Should Fail
```bash
# Try using the old refresh token again
curl -X POST http://localhost:8080/api/v1/auth/refresh-token \
  -H "Content-Type: application/json" \
  -d "{
    \"refresh_token\": \"$REFRESH_TOKEN\"
  }"
```

**Expected:**
```json
{
  "success": false,
  "message": "Invalid or expired refresh token",
  "error": {
    "refresh_token": "Please login again"
  }
}
```

**Status Code:** 401 Unauthorized

---

## Complete Test Script

Save as `test_jwt_blacklist.sh`:

```bash
#!/bin/bash

BASE_URL="http://localhost:8080/api/v1"
EMAIL="test@example.com"
PASSWORD="password123"

echo "=== Test 1: Login ==="
LOGIN_RESPONSE=$(curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
echo $LOGIN_RESPONSE | jq

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.token')
REFRESH_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.refresh_token')

echo -e "\n=== Test 2: Use Token ==="
curl -s $BASE_URL/auth/profile \
  -H "Authorization: Bearer $TOKEN" | jq

echo -e "\n=== Test 3: Logout ==="
curl -s -X POST $BASE_URL/auth/logout \
  -H "Authorization: Bearer $TOKEN" | jq

echo -e "\n=== Test 4: Try Using Blacklisted Token (Should Fail) ==="
curl -s $BASE_URL/auth/profile \
  -H "Authorization: Bearer $TOKEN" | jq

echo -e "\n=== Test 5: Login Again ==="
LOGIN_RESPONSE=$(curl -s -X POST $BASE_URL/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
echo $LOGIN_RESPONSE | jq

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.token')
REFRESH_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.refresh_token')

echo -e "\n=== Test 6: Refresh Token ==="
REFRESH_RESPONSE=$(curl -s -X POST $BASE_URL/auth/refresh-token \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
echo $REFRESH_RESPONSE | jq

NEW_TOKEN=$(echo $REFRESH_RESPONSE | jq -r '.data.token')
NEW_REFRESH=$(echo $REFRESH_RESPONSE | jq -r '.data.refresh_token')

echo -e "\n=== Test 7: Old Refresh Token Should Fail ==="
curl -s -X POST $BASE_URL/auth/refresh-token \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | jq

echo -e "\n=== Test 8: New Token Should Work ==="
curl -s $BASE_URL/auth/profile \
  -H "Authorization: Bearer $NEW_TOKEN" | jq

echo -e "\n=== All Tests Complete ==="
```

Run with:
```bash
chmod +x test_jwt_blacklist.sh
./test_jwt_blacklist.sh
```

---

## Redis Verification

Check all keys:
```bash
redis-cli
> KEYS *
> SCAN 0 MATCH jwt_blacklist:* COUNT 100
> SCAN 0 MATCH refresh_token:* COUNT 100
> SCAN 0 MATCH session:* COUNT 100
```

Monitor real-time:
```bash
redis-cli MONITOR
```

Check TTLs:
```bash
redis-cli
> TTL jwt_blacklist:a1b2c3d4...
86400  # Should be ~24 hours (JWT_EXPIRED)

> TTL refresh_token:abc123...
604800  # Should be ~7 days

> TTL session:sess_abc123...
172800  # Should be ~48 hours (SESSION_TTL)
```

---

## Expected Log Output

```
[INFO] Session created successfully
  user_id: 550e8400-e29b-41d4-a716-446655440000
  session_preview: sess_abc1...
  device_id: Chrome-Windows-10

[INFO] JWT blacklisted successfully
  user_id: 550e8400-e29b-41d4-a716-446655440000
  jti: a1b2c3d4... (preview)

[INFO] All refresh tokens deleted
  user_id: 550e8400-e29b-41d4-a716-446655440000

[INFO] Token refreshed successfully
  user_id: 550e8400-e29b-41d4-a716-446655440000
  session_id: sess_abc1... (preview)

[WARN] Invalid refresh token
  error: refresh token not found or expired
```

---

## Success Criteria

✅ Login returns both `token` and `refresh_token`  
✅ Access token works for authenticated requests  
✅ Logout blacklists JWT in Redis  
✅ Blacklisted token returns 401 "Token has been revoked"  
✅ Refresh token generates new tokens  
✅ Old refresh token becomes invalid after use  
✅ Redis keys have correct TTL  
✅ Session deleted on logout  
✅ All user refresh tokens deleted on logout  

---

## Troubleshooting

**Problem**: Token not blacklisted after logout  
**Check**: Logs for "JWT blacklisted successfully"  
**Fix**: Ensure JWT has valid claims with `jti`

**Problem**: Refresh token fails immediately  
**Check**: `redis-cli KEYS refresh_token:*`  
**Fix**: Verify Redis connection and token storage

**Problem**: Blacklisted token still works  
**Check**: Middleware blacklist check is enabled  
**Fix**: Verify `middleware/auth.go` has blacklist check

**Problem**: Redis keys not expiring  
**Check**: `redis-cli TTL <key>`  
**Fix**: Ensure TTL is set during key creation
