# 🔥 Redis Session Management - Complete Guide

## 📚 Daftar Isi

1. [Penjelasan Konsep](#-penjelasan-konsep)
2. [Arsitektur System](#-arsitektur-system)
3. [Alur Detail Login](#-alur-detail-login)
4. [Alur Detail Request Validation](#-alur-detail-request-validation)
5. [Struktur Data di Redis](#-struktur-data-di-redis)
6. [Code Breakdown](#-code-breakdown)
7. [Setup & Testing](#-setup--testing)
8. [Troubleshooting](#-troubleshooting)

---

## 🎯 Penjelasan Konsep

### Apa itu Session Management?

**Session** adalah cara untuk mengingat informasi user ketika mereka melakukan multiple requests ke server. Karena HTTP adalah **stateless** (setiap request independen), kita perlu cara untuk "mengingat" siapa user yang sedang login.

### Kenapa Pakai Redis?

| Fitur | Penjelasan |
|-------|------------|
| **In-Memory** | Super cepat (microseconds) karena data di RAM |
| **Key-Value Store** | Simple struktur: `key` → `value` (perfect untuk session) |
| **TTL (Time To Live)** | Auto-expire session setelah waktu tertentu |
| **Distributed** | Bisa diakses dari multiple servers (scalable) |
| **Persistence** | Bisa save to disk (optional) |

### Session ID vs JWT vs Redis

```
┌────────────────────────────────────────────────────────────┐
│                    AUTHENTICATION FLOW                      │
└────────────────────────────────────────────────────────────┘

User Login
    ↓
┌─────────────────────────────────────────────────────────────┐
│ 1. SERVER GENERATES                                         │
│    ├─ Session ID  : sess_abc123... (random 32 bytes)       │
│    ├─ Session Data: {user_id, ip, ua, purpose, ...}        │
│    └─ JWT Token   : eyJhbGci... (contains session_id)      │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. STORE IN REDIS                                           │
│    Key  : session:sess_abc123...                            │
│    Value: {"user_id": "...", "ip": "...", ...}              │
│    TTL  : 48 hours (auto-expire)                            │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. RETURN JWT TO CLIENT                                     │
│    {                                                         │
│      "token": "eyJhbGci...",  ← Contains session_id         │
│      "user": {...}                                           │
│    }                                                         │
└─────────────────────────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. CLIENT SAVES JWT                                         │
│    - LocalStorage / Cookie / Memory                         │
│    - Sends in every request: Authorization: Bearer <JWT>    │
└─────────────────────────────────────────────────────────────┘
```

---

## 🏗️ Arsitektur System

### Komponen Utama

```
┌──────────────────────────────────────────────────────────────┐
│                     ARCHITECTURE                              │
└──────────────────────────────────────────────────────────────┘

┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Client    │         │   Go Server  │         │    Redis    │
│  (Browser)  │         │   (Gin API)  │         │  (Storage)  │
└─────────────┘         └──────────────┘         └─────────────┘
      │                         │                         │
      │  1. POST /login         │                         │
      │──────────────────────>  │                         │
      │  {email, password}      │                         │
      │                         │                         │
      │                         │  2. Validate User       │
      │                         │     (Check MySQL)       │
      │                         │                         │
      │                         │  3. Generate SessionID  │
      │                         │     sess_abc123...      │
      │                         │                         │
      │                         │  4. Store Session       │
      │                         │─────────────────────>   │
      │                         │  SET session:sess_abc... │
      │                         │  {user_id, ip, ...}     │
      │                         │  EX 172800 (48h)        │
      │                         │                         │
      │                         │  5. Generate JWT        │
      │                         │     {session_id: "sess_..."}
      │                         │                         │
      │  6. Return JWT          │                         │
      │ <──────────────────────│                         │
      │  {token: "eyJhbGci..."}│                         │
      │                         │                         │
      │  7. GET /api/users      │                         │
      │──────────────────────>  │                         │
      │  Authorization: Bearer  │                         │
      │  eyJhbGci...            │                         │
      │                         │                         │
      │                         │  8. Decode JWT          │
      │                         │     Extract session_id  │
      │                         │                         │
      │                         │  9. Validate Session    │
      │                         │─────────────────────>   │
      │                         │  GET session:sess_abc...│
      │                         │                         │
      │                         │ 10. Session Data        │
      │                         │ <───────────────────────│
      │                         │  {user_id, ip, ...}     │
      │                         │                         │
      │                         │ 11. Check IP/UA Match   │
      │                         │     Refresh TTL         │
      │                         │                         │
      │ 12. Success Response    │                         │
      │ <──────────────────────│                         │
      │  {data: [...]}          │                         │
      │                         │                         │
```

---

## 🚀 Alur Detail Login

### Step by Step: User Login

```go
// File: controllers/auth/login.go
func (c *Controller) Login(ctx *gin.Context) {
```

#### **STEP 1: Validate Credentials**

```go
// 1.1 Parse request body
var loginReq dto.LoginRequest
if err := ctx.ShouldBindJSON(&loginReq); err != nil {
    return // Bad request
}

// 1.2 Get user from database (MySQL)
user, userAuth, err := c.repo.GetUserAuthRepository().GetUserForLogin(loginReq.EmailOrUsername)

// 1.3 Verify password
if err := utils.CheckPassword(userAuth.PasswordHash, loginReq.Password); err != nil {
    return // Invalid credentials
}
```

**Flow:**
```
Client Request
    ↓
┌────────────────────────────────────┐
│ Parse JSON                         │
│ {"email": "user@example.com",      │
│  "password": "secret123"}          │
└────────────────────────────────────┘
    ↓
┌────────────────────────────────────┐
│ Query MySQL Database               │
│ SELECT * FROM users                │
│ WHERE email = 'user@example.com'   │
└────────────────────────────────────┘
    ↓
┌────────────────────────────────────┐
│ Verify Password Hash               │
│ bcrypt.CompareHashAndPassword()    │
└────────────────────────────────────┘
    ↓
   ✅ Valid
```

---

#### **STEP 2: Generate Session ID**

```go
// 2.1 Get device info
deviceID, lastIP := clientip.GetDeviceAndIPInfo(ctx)
// deviceID = "fp_724_0_5fb"
// lastIP = "::1"

// 2.2 Create session in Redis
sessionID, err := middleware.CreateSession(
    context.Background(),
    user.ID,                    // "92aabb21-3597-4f3f-9fa5-bb82868f7039"
    "login",                    // purpose
    ctx.ClientIP(),             // "::1"
    ctx.GetHeader("User-Agent"), // "PostmanRuntime/7.48.0"
    *deviceID,                  // "fp_724_0_5fb"
)
```

**Flow:**
```
middleware.CreateSession()
    ↓
┌────────────────────────────────────────────────────────────┐
│ manager := GetSessionManager()                             │
│ // Get global Redis manager instance                       │
└────────────────────────────────────────────────────────────┘
    ↓
┌────────────────────────────────────────────────────────────┐
│ sess, err := manager.Create(ctx, userID, purpose, ...)     │
│ // Calls session manager's Create method                   │
└────────────────────────────────────────────────────────────┘
    ↓
┌────────────────────────────────────────────────────────────┐
│ INSIDE manager.Create():                                   │
│                                                             │
│ 1. Generate random session ID:                             │
│    randomBytes := make([]byte, 32)                         │
│    rand.Read(randomBytes)                                  │
│    sessionID = fmt.Sprintf("sess_%x", randomBytes)         │
│    // Result: sess_8f3a2b1c9e7d5f4a6b8c1e3d7f9a2b4c...    │
│                                                             │
│ 2. Create Session struct:                                  │
│    session := &Session{                                    │
│        ID:        sessionID,                               │
│        UserID:    "92aabb21-...",                          │
│        Purpose:   "login",                                 │
│        IPAddress: "::1",                                   │
│        UserAgent: "PostmanRuntime/7.48.0",                 │
│        DeviceID:  "fp_724_0_5fb",                          │
│        CreatedAt: time.Now(),                              │
│        ExpiresAt: time.Now().Add(48 * time.Hour),         │
│        LastSeen:  time.Now(),                              │
│    }                                                        │
│                                                             │
│ 3. Store in Redis:                                         │
│    m.store.Set(ctx, sessionID, session, 48*time.Hour)     │
└────────────────────────────────────────────────────────────┘
    ↓
┌────────────────────────────────────────────────────────────┐
│ INSIDE store.Set():                                        │
│                                                             │
│ 1. Marshal session to JSON:                                │
│    data, _ := json.Marshal(session)                        │
│    // Result:                                              │
│    // {"id":"sess_8f3a...","user_id":"92aa...", ...}       │
│                                                             │
│ 2. Create Redis key:                                       │
│    key := "session:sess_8f3a2b1c9e..."                    │
│                                                             │
│ 3. Store in Redis with TTL:                                │
│    s.client.Set(ctx, key, data, 48*time.Hour)             │
│    // Redis command:                                       │
│    // SET "session:sess_8f3a..." "{...}" EX 172800        │
└────────────────────────────────────────────────────────────┘
    ↓
┌────────────────────────────────────────────────────────────┐
│ REDIS DATABASE NOW CONTAINS:                               │
│                                                             │
│ Key: session:sess_8f3a2b1c9e7d5f4a6b8c1e3d7f9a2b4c...     │
│ Value: {                                                   │
│   "id": "sess_8f3a...",                                    │
│   "user_id": "92aabb21-3597-4f3f-9fa5-bb82868f7039",      │
│   "purpose": "login",                                      │
│   "ip_address": "::1",                                     │
│   "user_agent": "PostmanRuntime/7.48.0",                   │
│   "device_id": "fp_724_0_5fb",                             │
│   "created_at": "2025-01-20T10:00:00Z",                    │
│   "expires_at": "2025-01-22T10:00:00Z",                    │
│   "last_seen": "2025-01-20T10:00:00Z"                      │
│ }                                                           │
│ TTL: 172800 seconds (48 hours)                             │
└────────────────────────────────────────────────────────────┘
    ↓
   Return sessionID = "sess_8f3a2b1c9e..."
```

---

#### **STEP 3: Generate JWT Token**

```go
// 3. Generate JWT with session_id embedded
token, err := utils.GenerateJWT(
    user.ID,              // "92aabb21-..."
    sessionID,            // "sess_8f3a..."
    *deviceID,            // "fp_724_0_5fb"
    *lastIP,              // "::1"
    user.Username,        // "bakaban"
    user.Email,           // "adehusni37@gmail.com"
    user.Role,            // "admin"
    user.HasPremiumAccessAt(time.Now()), // true
    userAuth.IsEmailVerified, // true
)
```

**JWT Payload:**
```json
{
  "user_id": "92aabb21-3597-4f3f-9fa5-bb82868f7039",
  "session_id": "sess_8f3a2b1c9e7d5f4a6b8c1e3d7f9a2b4c8e1d3f5a7b9c1e3d5f7a9b1c3e5d7f9a",
  "device_id": "fp_724_0_5fb",
  "last_ip": "::1",
  "username": "bakaban",
  "email": "adehusni37@gmail.com",
  "role": "admin",
  "is_verified": true,
  "iss": "lihatin-go",
  "sub": "92aabb21-3597-4f3f-9fa5-bb82868f7039",
  "exp": 1759569704,
  "iat": 1759396904
}
```

**Encoded JWT:**
```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiOTJhYWJiMjEtMzU5Ny00ZjNmLTlmYTUtYmI4Mjg2OGY3MDM5Iiwic2Vzc2lvbl9pZCI6InNlc3NfOGYzYTJiMWM5ZTdkNWY0YTZiOGMxZTNkN2Y5YTJiNGM4ZTFkM2Y1YTdiOWMxZTNkNWY3YTliMWMzZTVkN2Y5YSIsImRldmljZV9pZCI6ImZwXzcyNF8wXzVmYiIsImxhc3RfaXAiOiI6OjEiLCJ1c2VybmFtZSI6ImJha2FiYW4iLCJlbWFpbCI6ImFkZWh1c25pMzdAZ21haWwuY29tIiwicm9sZSI6ImFkbWluIiwiaXNfcHJlbWl1bSI6dHJ1ZSwiaXNfdmVyaWZpZWQiOnRydWUsImlzcyI6ImxpaGF0aW4tZ28iLCJzdWIiOiI5MmFhYmIyMS0zNTk3LTRmM2YtOWZhNS1iYjgyODY4ZjcwMzkiLCJleHAiOjE3NTk1Njk3MDQsImlhdCI6MTc1OTM5NjkwNH0.abc123def456...
```

---

#### **STEP 4: Return Response to Client**

```go
// 4. Prepare response
responseData := map[string]interface{}{
    "user":          user,
    "token":         token,
    "refresh_token": refreshToken,
    "requires_2fa":  hasTOTP,
    "is_verified":   userAuth.IsEmailVerified,
}

ctx.JSON(http.StatusOK, common.APIResponse{
    Success: true,
    Data:    responseData,
    Message: "Login successful",
})
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "92aabb21-3597-4f3f-9fa5-bb82868f7039",
      "username": "bakaban",
      "email": "adehusni37@gmail.com",
      "role": "admin",
      "premium_access": {
        "status": "active",
        "tier": "premium"
      }
    },
    "token": "eyJhbGci...",
    "refresh_token": "eyJhbGci...",
    "requires_2fa": false,
    "is_verified": true
  },
  "message": "Login successful"
}
```

---

## 🔐 Alur Detail Request Validation

### Step by Step: Protected Route Access

```go
// File: middleware/auth.go
func AuthMiddleware(userRepo repositories.UserRepository) gin.HandlerFunc {
```

#### **STEP 1: Extract JWT from Request**

```go
// 1.1 Get Authorization header
authHeader := c.GetHeader("Authorization")
// "Bearer eyJhbGci..."

// 1.2 Extract token
token := utils.ExtractTokenFromHeader(authHeader)
// "eyJhbGci..."
```

**Client Request:**
```http
GET /api/users HTTP/1.1
Host: localhost:8080
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

---

#### **STEP 2: Validate JWT**

```go
// 2. Validate JWT and extract claims
claims, err := utils.ValidateJWT(token)
if err != nil {
    return // Invalid token
}
```

**Flow:**
```
utils.ValidateJWT(token)
    ↓
┌────────────────────────────────────┐
│ 1. Parse JWT                       │
│    jwt.ParseWithClaims()           │
└────────────────────────────────────┘
    ↓
┌────────────────────────────────────┐
│ 2. Verify Signature                │
│    Check HMAC-SHA256 signature     │
│    with JWT_SECRET                 │
└────────────────────────────────────┘
    ↓
┌────────────────────────────────────┐
│ 3. Check Expiration                │
│    if time.Now() > claims.exp {    │
│        return error                │
│    }                                │
└────────────────────────────────────┘
    ↓
┌────────────────────────────────────┐
│ 4. Extract Claims                  │
│    claims = {                      │
│      user_id: "92aabb21-...",      │
│      session_id: "sess_8f3a...",   │
│      username: "bakaban",          │
│      ...                            │
│    }                                │
└────────────────────────────────────┘
    ↓
   Return claims ✅
```

---

#### **STEP 3: Validate Session in Redis**

```go
// 3.1 Get session_id from JWT claims
sessionID := claims.SessionID
// "sess_8f3a2b1c9e..."

// 3.2 Validate session format
_, isValid := session.ValidateSessionID(sessionID)
if !isValid {
    return // Invalid format
}

// 3.3 Validate session with user/IP/UA check
validMetadata, err := session.ValidateSessionForUser(
    sessionID,
    claims.UserID,
    c.ClientIP(),
    c.GetHeader("User-Agent"),
)
```

**Flow:**
```
session.ValidateSessionForUser()
    ↓
┌────────────────────────────────────────────────────────────┐
│ 1. Check session ID format                                 │
│    if !strings.HasPrefix(sessionID, "sess_") { error }     │
│    if len(hexPart) != 64 { error }                         │
└────────────────────────────────────────────────────────────┘
    ↓
┌────────────────────────────────────────────────────────────┐
│ 2. Get global Redis manager                                │
│    manager := GetManager()                                 │
└────────────────────────────────────────────────────────────┘
    ↓
┌────────────────────────────────────────────────────────────┐
│ 3. Fetch session from Redis                                │
│    sess, err := manager.Get(context.Background(), sessionID)│
└────────────────────────────────────────────────────────────┘
    ↓
┌────────────────────────────────────────────────────────────┐
│ INSIDE manager.Get():                                      │
│                                                             │
│ 1. Fetch from Redis:                                       │
│    key := "session:sess_8f3a..."                          │
│    data, err := s.client.Get(ctx, key).Result()           │
│    // Redis command: GET "session:sess_8f3a..."           │
│                                                             │
│ 2. Check if found:                                         │
│    if err == redis.Nil {                                   │
│        return ErrSessionNotFound                           │
│    }                                                        │
│                                                             │
│ 3. Unmarshal JSON:                                         │
│    var session Session                                     │
│    json.Unmarshal(data, &session)                          │
│                                                             │
│ 4. Check expiration:                                       │
│    if time.Now().After(session.ExpiresAt) {               │
│        return ErrSessionExpired                            │
│    }                                                        │
└────────────────────────────────────────────────────────────┘
    ↓
┌────────────────────────────────────────────────────────────┐
│ 4. Validate user matches                                   │
│    if sess.UserID != expectedUserID {                      │
│        return error "session belongs to different user"    │
│    }                                                        │
└────────────────────────────────────────────────────────────┘
    ↓
┌────────────────────────────────────────────────────────────┐
│ 5. Check IP consistency (optional warning)                 │
│    if sess.IPAddress != currentIP {                        │
│        log.Warn("IP changed")                              │
│    }                                                        │
└────────────────────────────────────────────────────────────┘
    ↓
┌────────────────────────────────────────────────────────────┐
│ 6. Check User-Agent consistency (optional warning)         │
│    if sess.UserAgent != currentUserAgent {                 │
│        log.Warn("User-Agent changed")                      │
│    }                                                        │
└────────────────────────────────────────────────────────────┘
    ↓
┌────────────────────────────────────────────────────────────┐
│ 7. Convert Session to SessionMetadata                      │
│    metadata := &SessionMetadata{                           │
│        UserID:    sess.UserID,                             │
│        Purpose:   sess.Purpose,                            │
│        IssuedAt:  sess.CreatedAt.Unix(),                   │
│        ExpiresAt: sess.ExpiresAt.Unix(),                   │
│        IPAddress: sess.IPAddress,                          │
│        UserAgent: sess.UserAgent,                          │
│    }                                                        │
└────────────────────────────────────────────────────────────┘
    ↓
   Return metadata ✅
```

---

#### **STEP 4: Set Context & Continue**

```go
// 4. Set user information in context
c.Set("user_id", claims.UserID)
c.Set("username", claims.Username)
c.Set("email", claims.Email)
c.Set("role", claims.Role)
c.Set("session_metadata", validMetadata)
c.Set("session_purpose", validMetadata.Purpose)

c.Next() // Continue to handler
```

**Flow:**
```
┌────────────────────────────────────┐
│ Middleware validates ✅             │
│ - JWT signature valid              │
│ - JWT not expired                  │
│ - Session exists in Redis          │
│ - Session not expired              │
│ - User ID matches                  │
└────────────────────────────────────┘
    ↓
┌────────────────────────────────────┐
│ Set context variables:             │
│ c.Set("user_id", "92aabb21-...")   │
│ c.Set("username", "bakaban")       │
│ c.Set("session_metadata", {...})   │
└────────────────────────────────────┘
    ↓
┌────────────────────────────────────┐
│ c.Next()                           │
│ Continue to route handler          │
└────────────────────────────────────┘
    ↓
┌────────────────────────────────────┐
│ Handler can access:                │
│ userID := c.Get("user_id")         │
│ username := c.Get("username")      │
└────────────────────────────────────┘
```

---

## 📦 Struktur Data di Redis

### Redis Key Pattern

```
session:{session_id}
```

**Contoh:**
```
session:sess_8f3a2b1c9e7d5f4a6b8c1e3d7f9a2b4c8e1d3f5a7b9c1e3d5f7a9b1c3e5d7f9a
```

### Redis Value (JSON)

```json
{
  "id": "sess_8f3a2b1c9e7d5f4a6b8c1e3d7f9a2b4c8e1d3f5a7b9c1e3d5f7a9b1c3e5d7f9a",
  "user_id": "92aabb21-3597-4f3f-9fa5-bb82868f7039",
  "purpose": "login",
  "ip_address": "::1",
  "user_agent": "PostmanRuntime/7.48.0",
  "device_id": "fp_724_0_5fb",
  "created_at": "2025-01-20T10:00:00Z",
  "expires_at": "2025-01-22T10:00:00Z",
  "last_seen": "2025-01-20T10:00:00Z"
}
```

### Redis Commands

```bash
# Set session (done by Go code)
SET "session:sess_abc123..." "{...json...}" EX 172800

# Get session
GET "session:sess_abc123..."

# Check TTL (time remaining)
TTL "session:sess_abc123..."
# Returns: (integer) 172800  (seconds)

# Delete session (logout)
DEL "session:sess_abc123..."

# List all sessions
KEYS "session:*"

# List sessions for specific user (pattern matching)
SCAN 0 MATCH "session:*" COUNT 100
```

---

## 💻 Code Breakdown

### File Structure

```
utils/session/
├── redis_store.go      # Redis CRUD operations
├── manager.go          # Session lifecycle management
├── session.go          # Session ID generation & format validation
├── metadata.go         # SessionMetadata struct & helpers
└── validation.go       # Session validation functions

middleware/
├── session.go          # Session middleware & helpers
└── auth.go             # Auth middleware (uses session validation)

controllers/auth/
├── login.go            # Login handler (creates session)
└── register.go         # Register handler (creates session)
```

---

### 1. Redis Store (`utils/session/redis_store.go`)

**Purpose:** Low-level Redis operations

```go
type RedisStore struct {
    client *redis.Client
}

// Set stores session in Redis with TTL
func (s *RedisStore) Set(ctx context.Context, sessionID string, session *Session, ttl time.Duration) error {
    // 1. Create Redis key
    key := fmt.Sprintf("session:%s", sessionID)
    
    // 2. Marshal session to JSON
    data, err := json.Marshal(session)
    
    // 3. Store in Redis with expiration
    return s.client.Set(ctx, key, data, ttl).Err()
    // Redis command: SET "session:sess_abc..." "{...}" EX 172800
}

// Get retrieves session from Redis
func (s *RedisStore) Get(ctx context.Context, sessionID string) (*Session, error) {
    // 1. Create Redis key
    key := fmt.Sprintf("session:%s", sessionID)
    
    // 2. Get from Redis
    data, err := s.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return nil, ErrSessionNotFound
    }
    
    // 3. Unmarshal JSON to Session
    var session Session
    json.Unmarshal(data, &session)
    
    return &session, nil
}

// Delete removes session from Redis
func (s *RedisStore) Delete(ctx context.Context, sessionID string) error {
    key := fmt.Sprintf("session:%s", sessionID)
    return s.client.Del(ctx, key).Err()
    // Redis command: DEL "session:sess_abc..."
}

// Refresh extends session TTL
func (s *RedisStore) Refresh(ctx context.Context, sessionID string, ttl time.Duration) error {
    key := fmt.Sprintf("session:%s", sessionID)
    return s.client.Expire(ctx, key, ttl).Err()
    // Redis command: EXPIRE "session:sess_abc..." 172800
}
```

---

### 2. Session Manager (`utils/session/manager.go`)

**Purpose:** High-level session lifecycle management

```go
type Manager struct {
    store      *RedisStore
    defaultTTL time.Duration
}

// Global manager instance
var globalManager *Manager

// Create creates new session and stores in Redis
func (m *Manager) Create(ctx context.Context, userID, purpose, ipAddress, userAgent, deviceID string) (*Session, error) {
    // 1. Generate secure random session ID
    sessionID, err := m.generateSecureID()
    // Result: "sess_8f3a2b1c9e7d5f4a6b8c1e3d7f9a2b4c..."
    
    // 2. Create Session struct
    now := time.Now()
    session := &Session{
        ID:        sessionID,
        UserID:    userID,
        Purpose:   purpose,
        IPAddress: ipAddress,
        UserAgent: userAgent,
        DeviceID:  deviceID,
        CreatedAt: now,
        ExpiresAt: now.Add(m.defaultTTL), // 48 hours from now
        LastSeen:  now,
    }
    
    // 3. Store in Redis
    if err := m.store.Set(ctx, sessionID, session, m.defaultTTL); err != nil {
        return nil, err
    }
    
    return session, nil
}

// Get retrieves and validates session
func (m *Manager) Get(ctx context.Context, sessionID string) (*Session, error) {
    // 1. Get from Redis
    session, err := m.store.Get(ctx, sessionID)
    if err != nil {
        return nil, err
    }
    
    // 2. Check if expired
    if time.Now().After(session.ExpiresAt) {
        // Delete expired session
        m.store.Delete(ctx, sessionID)
        return nil, ErrSessionExpired
    }
    
    return session, nil
}

// Delete removes session (logout)
func (m *Manager) Delete(ctx context.Context, sessionID string) error {
    return m.store.Delete(ctx, sessionID)
}

// Refresh extends session lifetime
func (m *Manager) Refresh(ctx context.Context, sessionID string) error {
    return m.store.Refresh(ctx, sessionID, m.defaultTTL)
}

// generateSecureID generates cryptographically secure random ID
func (m *Manager) generateSecureID() (string, error) {
    // Generate 32 bytes of random data
    randomBytes := make([]byte, 32)
    rand.Read(randomBytes)
    
    // Convert to hex and add prefix
    return fmt.Sprintf("sess_%x", randomBytes), nil
    // Result: sess_8f3a2b1c9e7d5f4a6b8c1e3d7f9a2b4c...
}
```

---

### 3. Session Middleware (`middleware/session.go`)

**Purpose:** Initialize manager & provide helper functions

```go
// InitSessionManager initializes global manager
func InitSessionManager() error {
    // 1. Get config from environment
    redisAddr := utils.GetEnvOrDefault(utils.EnvRedisAddr, "localhost:6379")
    redisPassword := utils.GetEnvOrDefault(utils.EnvRedisPassword, "")
    redisDB := utils.GetEnvAsInt(utils.EnvRedisDB, 0)
    sessionTTLHours := utils.GetEnvAsInt(utils.EnvSessionTTL, 48)
    
    // 2. Create new manager
    manager, err := session.NewManager(
        redisAddr,
        redisPassword,
        redisDB,
        time.Duration(sessionTTLHours)*time.Hour,
    )
    
    // 3. Set as global manager
    session.SetManager(manager)
    
    return nil
}

// CreateSession is a helper to create session
func CreateSession(ctx context.Context, userID, purpose, ipAddress, userAgent, deviceID string) (string, error) {
    manager := GetSessionManager()
    sess, err := manager.Create(ctx, userID, purpose, ipAddress, userAgent, deviceID)
    if err != nil {
        return "", err
    }
    return sess.ID, nil
}
```

---

### 4. Auth Middleware (`middleware/auth.go`)

**Purpose:** Validate JWT + Session on each request

```go
func AuthMiddleware(userRepo repositories.UserRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Get JWT from header
        authHeader := c.GetHeader("Authorization")
        token := utils.ExtractTokenFromHeader(authHeader)
        
        // 2. Validate JWT
        claims, err := utils.ValidateJWT(token)
        if err != nil {
            c.JSON(http.StatusUnauthorized, ...)
            c.Abort()
            return
        }
        
        // 3. Validate session if present
        if claims.SessionID != "" {
            // Basic format validation
            _, isValid := session.ValidateSessionID(claims.SessionID)
            if !isValid {
                c.JSON(http.StatusUnauthorized, ...)
                c.Abort()
                return
            }
            
            // Advanced validation: check Redis + user/IP/UA
            validMetadata, err := session.ValidateSessionForUser(
                claims.SessionID,
                claims.UserID,
                c.ClientIP(),
                c.GetHeader("User-Agent"),
            )
            
            if err != nil {
                c.JSON(http.StatusUnauthorized, ...)
                c.Abort()
                return
            }
            
            // Add to context
            c.Set("session_metadata", validMetadata)
        }
        
        // 4. Set user info in context
        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)
        
        c.Next()
    }
}
```

---

## 🧪 Setup & Testing

### 1. Install Redis

#### **Option A: Docker (Recommended)**

```bash
# Pull image
docker pull redis:alpine

# Run Redis container
docker run -d \
  --name redis-session \
  -p 6379:6379 \
  redis:alpine

# Check if running
docker ps | grep redis

# Test connection
docker exec -it redis-session redis-cli ping
# Expected: PONG
```

#### **Option B: WSL (Windows Subsystem for Linux)**

```bash
# Open WSL
wsl

# Install Redis
sudo apt update
sudo apt install redis-server

# Start Redis
sudo service redis-server start

# Test
redis-cli ping
# Expected: PONG
```

#### **Option C: Native Windows**

Download from: https://github.com/microsoftarchive/redis/releases
- Download `Redis-x64-3.0.504.msi`
- Install
- Redis runs as Windows Service

---

### 2. Configure Environment

```env
# .env file
REDIS_ADDR="localhost:6379"
REDIS_PASSWORD=""
REDIS_DB="0"
SESSION_TTL_HOURS="48"
JWT_SECRET="your-secret-key"
JWT_EXPIRED="48"
```

---

### 3. Start Application

```bash
# Terminal 1: Start app
air
# Or
go run main.go
```

**Expected logs:**
```
[INFO] Session manager initialized redis_addr=localhost:6379 session_ttl_hours=48
[INFO] Server started on :8080
```

---

### 4. Test Login

```bash
# Terminal 2: Login request
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email_or_username": "bakaban",
    "password": "your_password"
  }'
```

**Expected response:**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "92aabb21-3597-4f3f-9fa5-bb82868f7039",
      "username": "bakaban",
      "email": "adehusni37@gmail.com"
    },
    "token": "eyJhbGci...",
    "refresh_token": "eyJhbGci..."
  },
  "message": "Login successful"
}
```

**Expected logs:**
```
[INFO] Session created successfully user_id=92aabb21-... session_preview=sess_8f3a2b1c... device_id=fp_724_0_5fb
```

---

### 5. Verify Redis Storage

```bash
# Access Redis CLI
docker exec -it redis-session redis-cli

# Or if using WSL/native
redis-cli
```

**Redis commands:**
```bash
# List all session keys
redis> KEYS session:*
1) "session:sess_8f3a2b1c9e7d5f4a6b8c1e3d7f9a2b4c8e1d3f5a7b9c1e3d5f7a9b1c3e5d7f9a"

# Get session data
redis> GET "session:sess_8f3a2b1c9e..."
"{\"id\":\"sess_8f3a...\",\"user_id\":\"92aabb21-...\",\"purpose\":\"login\",\"ip_address\":\"::1\",\"user_agent\":\"PostmanRuntime/7.48.0\",\"device_id\":\"fp_724_0_5fb\",\"created_at\":\"2025-01-20T10:00:00Z\",\"expires_at\":\"2025-01-22T10:00:00Z\",\"last_seen\":\"2025-01-20T10:00:00Z\"}"

# Check TTL (remaining time in seconds)
redis> TTL "session:sess_8f3a2b1c9e..."
(integer) 172795  # 48 hours - 5 seconds

# Check session type
redis> TYPE "session:sess_8f3a2b1c9e..."
string

# Get all info about the key
redis> OBJECT ENCODING "session:sess_8f3a2b1c9e..."
embstr
```

---

### 6. Test Protected Route

```bash
# Use the token from login response
curl -X GET http://localhost:8080/api/users \
  -H "Authorization: Bearer eyJhbGci..."
```

**Expected logs:**
```
[INFO] Session validated successfully user_id=92aabb21-... session_preview=sess_8f3a2b1c... purpose=login
```

---

### 7. Test Logout (Delete Session)

```go
// In logout handler:
middleware.DeleteSession(context.Background(), sessionID)
```

```bash
# Verify session deleted
redis> GET "session:sess_8f3a2b1c9e..."
(nil)  # Session not found
```

---

## 🐛 Troubleshooting

### Problem 1: "Failed to create session in Redis"

**Symptoms:**
```
[ERROR] Failed to create session in Redis user_id=... error=dial tcp 127.0.0.1:6379: connect: connection refused
```

**Solution:**

1. **Check Redis is running:**
```bash
# Docker
docker ps | grep redis

# WSL
wsl sudo service redis-server status

# Windows
Get-Service Redis
```

2. **Start Redis if not running:**
```bash
# Docker
docker start redis-session

# WSL
wsl sudo service redis-server start

# Windows
net start Redis
```

3. **Test Redis connection:**
```bash
# Docker
docker exec -it redis-session redis-cli ping

# Direct
redis-cli ping
```

---

### Problem 2: "Session not found or expired"

**Symptoms:**
```
[WARN] Session validation failed user_id=... error=session not found
```

**Causes:**
1. Session expired (TTL reached 0)
2. Redis was flushed/restarted
3. Wrong session ID

**Debug:**
```bash
# Check if session exists
redis-cli KEYS "session:*"

# Check TTL
redis-cli TTL "session:sess_abc..."
# If returns -2: key doesn't exist
# If returns -1: key exists but no TTL
# If returns positive number: seconds remaining
```

---

### Problem 3: "IP address changed during session"

**Symptoms:**
```
[WARN] IP address changed during session user_id=... original_ip=192.168.1.100 current_ip=192.168.1.101
```

**Explanation:**
- User's IP changed (mobile switching networks, VPN, etc.)
- This is just a **warning**, not an error
- Session still works (optional security check)

**For strict IP validation:**
```go
// Use ValidateSessionStrict instead
validMetadata, err := session.ValidateSessionStrict(
    sessionID,
    expectedUserID,
    currentIP,
    currentUserAgent,
)
// This will return error if IP changes
```

---

### Problem 4: Session masuk ke database (MySQL) bukan Redis

**Penjelasan:**
- **MySQL** = Persistent storage untuk users, posts, etc.
- **Redis** = In-memory cache untuk sessions (temporary data)
- Session **TIDAK** masuk ke MySQL
- Session **HANYA** di Redis

**Cara cek Redis (bukan Navicat):**

1. **Redis CLI:**
```bash
redis-cli KEYS "session:*"
```

2. **RedisInsight (GUI):**
- Download: https://redis.com/redis-enterprise/redis-insight/
- Connect to localhost:6379
- Browse keys visually

3. **Another Redis Desktop Manager:**
- https://github.com/qishibo/AnotherRedisDesktopManager
- Free, cross-platform

---

### Problem 5: "do not pass a nil Context"

**Error:**
```go
do not pass a nil Context, even if a function permits it
```

**Solution:**
```go
// ❌ Wrong
manager.Get(nil, sessionID)

// ✅ Correct
manager.Get(context.Background(), sessionID)

// ✅ Or use request context
manager.Get(ctx.Request.Context(), sessionID)
```

---

## 📊 Redis Monitoring

### Real-time Monitoring

```bash
# Monitor all Redis commands in real-time
redis-cli MONITOR

# Output example:
# 1642680000.123456 [0 172.17.0.1:12345] "SET" "session:sess_abc..." "{...}" "EX" "172800"
# 1642680001.456789 [0 172.17.0.1:12346] "GET" "session:sess_abc..."
```

### Session Statistics

```bash
# Count total sessions
redis-cli KEYS "session:*" | wc -l

# Get Redis memory usage
redis-cli INFO memory

# Get session key sizes
redis-cli --bigkeys

# Get detailed info
redis-cli INFO all
```

### Debugging Commands

```bash
# Check connection
redis-cli PING

# Select database (if not using default 0)
redis-cli -n 0

# Scan sessions (better than KEYS for production)
redis-cli SCAN 0 MATCH "session:*" COUNT 100

# Delete all sessions (careful!)
redis-cli KEYS "session:*" | xargs redis-cli DEL

# Flush specific database
redis-cli -n 0 FLUSHDB

# Flush all databases (careful!)
redis-cli FLUSHALL
```

---

## 🎓 Best Practices

### ✅ DO

1. **Always set TTL** - Auto-expire prevents dead sessions
2. **Refresh on activity** - Extend TTL when user is active
3. **Validate IP/UA** - Security check for suspicious changes
4. **Log session events** - Track creation, validation, deletion
5. **Use context** - Pass `context.Background()` or request context
6. **Handle errors** - Session not found ≠ invalid credentials

### ❌ DON'T

1. **Don't skip Redis validation** - JWT alone is not enough
2. **Don't store sensitive data** - Keep passwords out of session
3. **Don't use predictable IDs** - Always use crypto/rand
4. **Don't set TTL too long** - Balance UX vs security
5. **Don't forget logout** - Delete session on logout
6. **Don't use KEYS in production** - Use SCAN instead (KEYS blocks Redis)

---

## 🚀 Performance Tips

### Redis Connection Pooling

```go
// Already handled by redis.NewClient()
client := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    PoolSize: 10, // Connection pool size
    MinIdleConns: 5, // Minimum idle connections
})
```

### Batch Operations

```go
// Get multiple sessions at once
pipe := s.client.Pipeline()
for _, sessionID := range sessionIDs {
    pipe.Get(ctx, fmt.Sprintf("session:%s", sessionID))
}
results, _ := pipe.Exec(ctx)
```

### Session Cleanup

```go
// Automatic: Redis handles TTL expiration
// Manual cleanup (if needed):
func (m *Manager) CleanupExpiredSessions() {
    // Redis automatically removes expired keys
    // No manual cleanup needed!
}
```

---

## 📝 Summary

### Session Lifecycle

```
1. Login → Generate Session ID → Store in Redis → Return JWT
2. Request → Extract JWT → Validate JWT → Check Redis → Process
3. Logout → Delete from Redis → JWT becomes invalid
```

### Key Points

| Aspect | Detail |
|--------|--------|
| **Session ID** | Random 32 bytes hex (sess_abc...) |
| **Storage** | Redis (in-memory, fast) |
| **TTL** | 48 hours (configurable) |
| **Security** | 256-bit entropy, unpredictable |
| **Validation** | Format check + Redis lookup + metadata check |
| **Expiration** | Automatic via Redis TTL |
| **Management** | Create, Get, Delete, Refresh, List |

### Data Flow

```
User → JWT (contains session_id) → Server → Redis (session data) → Validate → Allow/Deny
```

### Files Modified

1. ✅ `controllers/auth/login.go` - Create session on login
2. ✅ `controllers/auth/register.go` - Create session on register
3. ✅ `middleware/auth.go` - Validate session on requests
4. ✅ `middleware/session.go` - Session helpers
5. ✅ `utils/session/*.go` - Session management logic

---

## 🎉 Conclusion

Sekarang kamu punya **production-grade session management** dengan:

- ✅ Cryptographically secure session IDs
- ✅ Fast Redis storage (in-memory)
- ✅ Automatic expiration (TTL)
- ✅ IP/User-Agent validation
- ✅ Multi-device support
- ✅ Easy logout (delete from Redis)
- ✅ Scalable (distributed sessions)

**Next Steps:**
1. Install Redis
2. Test login endpoint
3. Verify session in Redis
4. Deploy & monitor

Happy coding! 🚀
