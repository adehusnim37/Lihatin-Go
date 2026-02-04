# Login Attempts API - Quick Reference

## 📋 Overview

API untuk monitoring dan tracking login attempts dengan support untuk filtering, pagination, dan analytics. Support akses untuk user (own attempts) dan admin (all attempts).

---

## 🔑 Authentication

**User Routes** (`/auth/login-attempts`):
- Require: `AuthMiddleware` (authenticated users)
- Access: User's own login attempts only

**Admin Routes** (`/auth/admin/login-attempts`):
- Require: `AuthMiddleware` + `AdminAuth` 
- Access: All login attempts (system-wide)

---

## 📌 Endpoints Summary

| Method | Endpoint | Access | Description |
|--------|----------|--------|-------------|
| GET | `/auth/login-attempts` | User/Admin | Get paginated login attempts with filters |
| GET | `/auth/login-attempts/:id` | User/Admin | Get single login attempt by ID |
| GET | `/auth/login-attempts/stats/:email_or_username/:days` | User/Admin | Get login statistics for user |
| GET | `/auth/login-attempts/recent-activity` | User/Admin | Get 24h activity summary |
| GET | `/auth/login-attempts/attempts-by-hour` | User/Admin | Get attempts grouped by hour |
| GET | `/auth/admin/login-attempts/top-failed-ips` | Admin Only | Get top IPs with failed attempts |
| GET | `/auth/admin/login-attempts/suspicious-activity` | Admin Only | Get suspicious login patterns |

---

## 1️⃣ Get Login Attempts (Paginated)

### Endpoint
```
GET /auth/login-attempts
GET /auth/admin/login-attempts
```

### Query Parameters

#### Pagination
| Parameter | Type | Default | Validation | Description |
|-----------|------|---------|------------|-------------|
| `page` | int | 1 | >= 1 | Page number |
| `limit` | int | 50 | User: 1-100<br>Admin: 1-500 | Items per page |
| `sort` | string | created_at | created_at, updated_at, email_or_username, ip_address, success | Sort field |
| `order_by` | string | desc | asc, desc | Sort order |

#### Filters
| Parameter | Type | Validation | Description |
|-----------|------|------------|-------------|
| `success` | boolean | - | Filter by success/failure |
| `id` | string | - | Exact ID match |
| `search` | string | >= 2 chars | Partial match on email/username |
| `username` | string | - | Exact username match |
| `ip_address` | string | 7-45 chars | Filter by IP address |
| `from_date` | string | RFC3339 format | Start date (inclusive) |
| `to_date` | string | RFC3339 format | End date (inclusive) |

### Request Example
```bash
# User - get own failed attempts (last 7 days)
curl -X GET "http://localhost:8080/auth/login-attempts?success=false&from_date=2026-01-28T00:00:00Z&page=1&limit=20" \
  -H "Cookie: token=your_token_here"

# Admin - search all attempts by IP
curl -X GET "http://localhost:8080/auth/admin/login-attempts?ip_address=192.168.1.100&sort=created_at&order_by=desc" \
  -H "Cookie: token=admin_token_here"

# Admin - search by username pattern
curl -X GET "http://localhost:8080/auth/admin/login-attempts?search=john&limit=50" \
  -H "Cookie: token=admin_token_here"
```

### Response
```json
{
  "success": true,
  "message": "Successfully retrieved login attempts",
  "data": {
    "attempts": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "email_or_username": "john@example.com",
        "ip_address": "192.168.1.100",
        "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        "success": false,
        "fail_reason": "Invalid password",
        "created_at": "2026-02-04T10:30:00Z",
        "updated_at": "2026-02-04T10:30:00Z"
      }
    ],
    "total_count": 150,
    "page": 1,
    "limit": 20,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false
  }
}
```

---

## 2️⃣ Get Single Login Attempt

### Endpoint
```
GET /auth/login-attempts/:id
GET /auth/admin/login-attempts/:id
```

### URL Parameters
- `id` (required): Login attempt UUID

### Authorization
- **Users**: Can only access their own attempts (matched by email)
- **Admins**: Can access any attempt

### Request Example
```bash
curl -X GET "http://localhost:8080/auth/login-attempts/550e8400-e29b-41d4-a716-446655440000" \
  -H "Cookie: token=your_token_here"
```

### Response
```json
{
  "success": true,
  "message": "Login attempt retrieved successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email_or_username": "john@example.com",
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0...",
    "success": false,
    "fail_reason": "Invalid password",
    "created_at": "2026-02-04T10:30:00Z",
    "updated_at": "2026-02-04T10:30:00Z"
  }
}
```

### Error Response (403 Forbidden)
```json
{
  "success": false,
  "error_code": "ACCESS_DENIED",
  "message": "You can only view your own login attempts"
}
```

---

## 3️⃣ Get Login Statistics

### Endpoint
```
GET /auth/login-attempts/stats/:email_or_username/:days
GET /auth/admin/login-attempts/stats/:email_or_username/:days
```

### URL Parameters
- `email_or_username` (required): User email or username
- `days` (required): Number of days to analyze (e.g., 7, 30)

### Authorization
- **Users**: Can only view their own stats
- **Admins**: Can view any user's stats

### Request Example
```bash
# User viewing own stats (last 7 days)
curl -X GET "http://localhost:8080/auth/login-attempts/stats/john@example.com/7" \
  -H "Cookie: token=your_token_here"

# Admin viewing any user stats (last 30 days)
curl -X GET "http://localhost:8080/auth/admin/login-attempts/stats/alice@example.com/30" \
  -H "Cookie: token=admin_token_here"
```

### Response
```json
{
  "success": true,
  "message": "Successfully retrieved login statistics",
  "data": {
    "stats": {
      "total_attempts": 45,
      "successful_attempts": 38,
      "failed_attempts": 7,
      "success_rate": 84.44,
      "last_successful_login": "2026-02-04T09:15:00Z"
    },
    "email_or_username": "john@example.com",
    "days": 7,
    "period_start": "2026-01-28T00:00:00Z",
    "period_end": "2026-02-04T12:00:00Z"
  }
}
```

---

## 4️⃣ Get Recent Activity Summary

### Endpoint
```
GET /auth/login-attempts/recent-activity
GET /auth/admin/login-attempts/recent-activity
```

### Description
Returns summary of login activity in the last 24 hours.

### Authorization
- **Users**: See only their own activity
- **Admins**: See system-wide activity

### Request Example
```bash
curl -X GET "http://localhost:8080/auth/login-attempts/recent-activity" \
  -H "Cookie: token=your_token_here"
```

### Response
```json
{
  "success": true,
  "message": "Successfully retrieved recent activity",
  "data": {
    "total_attempts": 15,
    "successful_attempts": 12,
    "failed_attempts": 3,
    "unique_ips": 2,
    "hours": 24,
    "since": "2026-02-03T12:00:00Z",
    "period_start": "2026-02-03T12:00:00Z",
    "period_end": "2026-02-04T12:00:00Z"
  }
}
```

---

## 5️⃣ Get Attempts by Hour

### Endpoint
```
GET /auth/login-attempts/attempts-by-hour
GET /auth/admin/login-attempts/attempts-by-hour
```

### Description
Returns login attempts grouped by hour of day (last 7 days).

### Authorization
- **Users**: See only their own attempts
- **Admins**: See all attempts

### Request Example
```bash
curl -X GET "http://localhost:8080/auth/admin/login-attempts/attempts-by-hour" \
  -H "Cookie: token=admin_token_here"
```

### Response
```json
{
  "success": true,
  "message": "Successfully retrieved attempts by hour",
  "data": {
    "attempts_by_hour": [
      {
        "hour": 0,
        "total_count": 5,
        "success_count": 4,
        "failed_count": 1
      },
      {
        "hour": 1,
        "total_count": 2,
        "success_count": 2,
        "failed_count": 0
      },
      {
        "hour": 9,
        "total_count": 45,
        "success_count": 40,
        "failed_count": 5
      }
    ],
    "days": 7
  }
}
```

---

## 6️⃣ Get Top Failed IPs (Admin Only)

### Endpoint
```
GET /auth/admin/login-attempts/top-failed-ips
```

### Description
Returns top 10 IP addresses with most failed login attempts.

### Request Example
```bash
curl -X GET "http://localhost:8080/auth/admin/login-attempts/top-failed-ips" \
  -H "Cookie: token=admin_token_here"
```

### Response
```json
{
  "success": true,
  "message": "Successfully retrieved top failed IPs",
  "data": {
    "top_failed_ips": [
      {
        "ip_address": "192.168.1.100",
        "failed_count": 25,
        "last_attempt": "2026-02-04T11:30:00Z"
      },
      {
        "ip_address": "10.0.0.50",
        "failed_count": 18,
        "last_attempt": "2026-02-04T10:15:00Z"
      }
    ],
    "limit": 10
  }
}
```

---

## 7️⃣ Get Suspicious Activity (Admin Only)

### Endpoint
```
GET /auth/admin/login-attempts/suspicious-activity
```

### Description
Returns potentially suspicious login patterns (>5 failed attempts in last hour).

### Risk Levels
- **critical**: >20 failed attempts OR >10 unique emails attempted
- **high**: >10 failed attempts OR >5 unique emails attempted  
- **medium**: 6-10 failed attempts

### Request Example
```bash
curl -X GET "http://localhost:8080/auth/admin/login-attempts/suspicious-activity" \
  -H "Cookie: token=admin_token_here"
```

### Response
```json
{
  "success": true,
  "message": "Successfully retrieved suspicious activity",
  "data": {
    "suspicious_activity": [
      {
        "ip_address": "203.0.113.45",
        "failed_count": 23,
        "last_attempt": "2026-02-04T11:59:00Z",
        "emails_attempted": 12,
        "risk_level": "critical"
      },
      {
        "ip_address": "198.51.100.78",
        "failed_count": 8,
        "last_attempt": "2026-02-04T11:45:00Z",
        "emails_attempted": 3,
        "risk_level": "medium"
      }
    ]
  }
}
```

---

## 🚨 Error Responses

### Validation Errors (400)
```json
{
  "success": false,
  "error_code": "VALIDATION_ERROR",
  "message": "Invalid pagination parameters",
  "field": "pagination",
  "details": {
    "page": "Page must be a positive integer",
    "limit": "Limit must be between 1 and 100"
  }
}
```

### Filter Errors (400)
```json
{
  "success": false,
  "error_code": "FILTER_ERROR",
  "message": "Invalid filter parameters",
  "field": "filters",
  "details": {
    "search": "Search query must be at least 2 characters",
    "ip_address": "Invalid IP address format"
  }
}
```

### Access Denied (403)
```json
{
  "success": false,
  "error_code": "ACCESS_DENIED",
  "message": "You can only view your own login attempts"
}
```

### Admin Only (403)
```json
{
  "success": false,
  "error_code": "ADMIN_ONLY",
  "message": "This endpoint requires admin privileges"
}
```

### Not Found (404)
```json
{
  "success": false,
  "error_code": "LOGIN_ATTEMPT_NOT_FOUND",
  "message": "Login attempt not found"
}
```

---

## 📊 Use Cases & Examples

### 1. User Monitoring Own Security
```bash
# Check recent failed logins
curl "http://localhost:8080/auth/login-attempts?success=false&limit=10"

# View login history from specific date
curl "http://localhost:8080/auth/login-attempts?from_date=2026-01-01T00:00:00Z"

# Get weekly statistics
curl "http://localhost:8080/auth/login-attempts/stats/myemail@example.com/7"
```

### 2. Admin Security Monitoring
```bash
# Find all failed logins from suspicious IP
curl "http://localhost:8080/auth/admin/login-attempts?ip_address=203.0.113.45&success=false"

# Search for attempts targeting specific user
curl "http://localhost:8080/auth/admin/login-attempts?username=admin"

# Get suspicious activity alerts
curl "http://localhost:8080/auth/admin/login-attempts/suspicious-activity"

# View peak login hours
curl "http://localhost:8080/auth/admin/login-attempts/attempts-by-hour"
```

### 3. Audit & Compliance
```bash
# Export all login attempts for date range (paginated)
curl "http://localhost:8080/auth/admin/login-attempts?from_date=2026-01-01T00:00:00Z&to_date=2026-01-31T23:59:59Z&limit=500"

# Find login attempts during specific time window
curl "http://localhost:8080/auth/admin/login-attempts?from_date=2026-02-04T00:00:00Z&to_date=2026-02-04T05:00:00Z"
```

---

## 🔧 Query Parameter Combinations

### Example Queries

#### 1. Failed Logins from Specific IP (Last 24h)
```
?ip_address=192.168.1.100&success=false&from_date=2026-02-03T12:00:00Z
```

#### 2. Successful Logins, Sorted by Email
```
?success=true&sort=email_or_username&order_by=asc&limit=100
```

#### 3. Search User Activity by Pattern
```
?search=john&from_date=2026-01-01T00:00:00Z&sort=created_at&order_by=desc
```

#### 4. Recent Failed Attempts (Most Recent First)
```
?success=false&sort=created_at&order_by=desc&limit=50
```

---

## 📝 Notes

1. **Date Format**: All dates must be in RFC3339 format: `YYYY-MM-DDTHH:MM:SSZ` or with timezone offset `YYYY-MM-DDTHH:MM:SS±HH:MM`

2. **Rate Limits**: 
   - User endpoints: Max 100 items per page
   - Admin endpoints: Max 500 items per page

3. **Authorization**:
   - Non-admin users automatically filtered to see only their own attempts
   - Cannot be bypassed by query parameters

4. **Performance**:
   - Use pagination for large datasets
   - Date range filters improve query performance
   - Indexed fields: `created_at`, `email_or_username`, `ip_address`, `success`

5. **Security**:
   - All endpoints require authentication
   - Admin endpoints require admin role
   - Sensitive data (passwords) never included in responses
   - User isolation enforced at repository level

---

## 🔗 Related Documentation

- [Authentication System](./AUTHENTICATION_SYSTEM_DOCS.md)
- [API Key Implementation](./API_KEY_DOCUMENTATION.md)
- [Logger API Reference](./LOGGER_API_QUICK_REFERENCE.md)
- [Error Handling Guide](./ERROR_HANDLING_MIGRATION_GUIDE.md)

---

**Last Updated**: February 4, 2026
