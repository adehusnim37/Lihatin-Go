# Logger API Quick Reference

## Endpoints

### 1. Get All Logs
```
GET /api/logs
```

**Query Parameters:**
- `page` (optional, default: 1) - Page number
- `limit` (optional, default: 10, max: 100) - Items per page
- `sort` (optional, default: created_at) - Sort field
- `order_by` (optional, default: desc) - Sort direction (asc/desc)

**Sort Fields:**
`created_at`, `updated_at`, `timestamp`, `level`, `action`, `status_code`, `response_time`, `username`, `method`, `route`

**Example:**
```bash
curl "http://localhost:8080/api/logs?page=1&limit=20&sort=timestamp&order_by=desc"
```

---

### 2. Get Logs by Username
```
GET /api/logs/user/:username
```

**Path Parameters:**
- `username` (required) - Username to filter by

**Query Parameters:** Same as "Get All Logs"

**Example:**
```bash
curl "http://localhost:8080/api/logs/user/john_doe?page=1&limit=10"
```

---

### 3. Get Logs by Short Link
```
GET /api/logs/short/:code
```

**Path Parameters:**
- `code` (required) - Short link code

**Query Parameters:** Same as "Get All Logs"

**Example:**
```bash
curl "http://localhost:8080/api/logs/short/abc123?page=1&limit=50"
```

---

### 4. Get Logs with Advanced Filtering
```
GET /api/logs/filter
```

**Query Parameters:**
- **Pagination:** `page`, `limit`, `sort`, `order_by` (same as above)
- **Filters:**
  - `username` - Filter by username
  - `action` - Filter by action (login, logout, create, etc.)
  - `method` - Filter by HTTP method (GET, POST, PUT, DELETE, etc.)
  - `route` - Filter by route (partial match)
  - `level` - Filter by log level (debug, info, warn, error, fatal)
  - `status_code` - Filter by HTTP status code
  - `ip_address` - Filter by IP address
  - `date_from` - Filter from date (format: 2006-01-02T15:04:05Z)
  - `date_to` - Filter to date (format: 2006-01-02T15:04:05Z)
  - `api_key` - Filter by API key

**Examples:**
```bash
# Get error logs
curl "http://localhost:8080/api/logs/filter?level=error&page=1"

# Get logs for specific user and action
curl "http://localhost:8080/api/logs/filter?username=john_doe&action=login&page=1"

# Get 500 errors in date range
curl "http://localhost:8080/api/logs/filter?status_code=500&date_from=2026-01-01T00:00:00Z&date_to=2026-02-01T00:00:00Z&page=1"

# Get POST requests from specific IP
curl "http://localhost:8080/api/logs/filter?method=POST&ip_address=192.168.1.100&page=1"

# Combine multiple filters
curl "http://localhost:8080/api/logs/filter?username=john&level=error&status_code=500&route=/api/users&page=1&limit=25&sort=timestamp&order_by=desc"
```

---

## Response Format

All endpoints return the same standardized format:

```json
{
  "success": true,
  "data": {
    "logs": [
      {
        "id": "uuid-v7-string",
        "level": "error",
        "message": "Failed to process request",
        "username": "john_doe",
        "timestamp": "2026-02-02T10:30:00Z",
        "ip_address": "192.168.1.100",
        "user_agent": "Mozilla/5.0...",
        "browser_info": "Chrome 120.0",
        "action": "api_request",
        "route": "/api/users/123",
        "method": "GET",
        "status_code": 500,
        "response_time": 1234,
        "created_at": "2026-02-02T10:30:00Z",
        "updated_at": "2026-02-02T10:30:00Z"
      }
    ],
    "total_count": 1523,
    "page": 1,
    "limit": 10,
    "total_pages": 153,
    "has_next": true,
    "has_prev": false
  },
  "message": "Logs retrieved successfully",
  "error": null
}
```

---

## Error Response Format

```json
{
  "success": false,
  "data": null,
  "message": "Invalid pagination parameters",
  "error": {
    "page": "Page must be a positive integer",
    "limit": "Limit must be between 1 and 100"
  }
}
```

---

## Common Use Cases

### 1. Build a Log Viewer with Pagination
```typescript
async function fetchLogs(page: number) {
  const response = await fetch(`/api/logs?page=${page}&limit=20`);
  const data = await response.json();
  
  return {
    logs: data.data.logs,
    hasNext: data.data.has_next,
    hasPrev: data.data.has_prev,
    totalPages: data.data.total_pages
  };
}
```

### 2. Monitor Error Logs
```typescript
async function fetchErrorLogs() {
  const response = await fetch('/api/logs/filter?level=error&limit=50&sort=timestamp&order_by=desc');
  const data = await response.json();
  return data.data.logs;
}
```

### 3. User Activity Audit
```typescript
async function getUserActivity(username: string, dateFrom: string, dateTo: string) {
  const params = new URLSearchParams({
    username,
    date_from: dateFrom,
    date_to: dateTo,
    page: '1',
    limit: '100',
    sort: 'timestamp',
    order_by: 'desc'
  });
  
  const response = await fetch(`/api/logs/filter?${params}`);
  const data = await response.json();
  return data.data;
}
```

### 4. API Health Monitoring
```typescript
async function checkAPIErrors(route: string) {
  const response = await fetch(`/api/logs/filter?route=${route}&status_code=500&limit=10`);
  const data = await response.json();
  
  return {
    errorCount: data.data.total_count,
    recentErrors: data.data.logs
  };
}
```

---

## Tips & Best Practices

### ✅ Do's

- **Use appropriate page sizes:** Start with 10-20 items per page
- **Cache results:** Cache frequently accessed pages
- **Use filters:** Combine filters to reduce data transfer
- **Sort strategically:** Sort by indexed fields for best performance
- **Handle pagination:** Always check `has_next` before fetching next page

### ❌ Don'ts

- **Don't use limit > 100:** Server enforces max limit of 100
- **Don't deep paginate:** Avoid page numbers > 1000 (use filters instead)
- **Don't ignore errors:** Always check `success` field in response
- **Don't skip validation:** Validate pagination parameters client-side too
- **Don't use leading wildcards in route filter:** Avoid `/logs/filter?route=%xyz` (slow)

---

## Performance Tips

1. **Use Indexes:** Ensure database indexes are created (see LOGGER_DATABASE_INDEXING_GUIDE.md)
2. **Filter First:** Apply filters to reduce result set before pagination
3. **Small Limits:** Use smaller limit values for faster responses
4. **Indexed Sorts:** Sort by indexed fields (created_at, timestamp, username, etc.)
5. **Date Ranges:** Use date_from/date_to to limit time window

---

## Postman Collection

Import this into Postman for quick testing:

```json
{
  "info": {
    "name": "Logger API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Get All Logs",
      "request": {
        "method": "GET",
        "url": {
          "raw": "{{base_url}}/api/logs?page=1&limit=10&sort=created_at&order_by=desc",
          "host": ["{{base_url}}"],
          "path": ["api", "logs"],
          "query": [
            {"key": "page", "value": "1"},
            {"key": "limit", "value": "10"},
            {"key": "sort", "value": "created_at"},
            {"key": "order_by", "value": "desc"}
          ]
        }
      }
    },
    {
      "name": "Get Logs by Username",
      "request": {
        "method": "GET",
        "url": {
          "raw": "{{base_url}}/api/logs/user/:username?page=1&limit=10",
          "host": ["{{base_url}}"],
          "path": ["api", "logs", "user", ":username"],
          "variable": [{"key": "username", "value": "john_doe"}]
        }
      }
    },
    {
      "name": "Get Error Logs",
      "request": {
        "method": "GET",
        "url": {
          "raw": "{{base_url}}/api/logs/filter?level=error&page=1&limit=50",
          "host": ["{{base_url}}"],
          "path": ["api", "logs", "filter"],
          "query": [
            {"key": "level", "value": "error"},
            {"key": "page", "value": "1"},
            {"key": "limit", "value": "50"}
          ]
        }
      }
    }
  ]
}
```

---

## Testing Examples

### cURL Examples

```bash
# Basic pagination
curl -X GET "http://localhost:8080/api/logs?page=1&limit=10"

# Sort by different field
curl -X GET "http://localhost:8080/api/logs?sort=status_code&order_by=asc"

# User-specific logs
curl -X GET "http://localhost:8080/api/logs/user/john_doe?page=1&limit=20"

# Filter error logs
curl -X GET "http://localhost:8080/api/logs/filter?level=error&limit=50"

# Complex filter
curl -X GET "http://localhost:8080/api/logs/filter?username=john&status_code=500&date_from=2026-01-01T00:00:00Z&page=1"

# Pretty print JSON
curl -X GET "http://localhost:8080/api/logs?page=1&limit=5" | jq .
```

---

## Support

For issues or questions:
1. Check [LOGGER_REFACTORING_SUMMARY.md](./LOGGER_REFACTORING_SUMMARY.md) for detailed documentation
2. Review [LOGGER_DATABASE_INDEXING_GUIDE.md](./LOGGER_DATABASE_INDEXING_GUIDE.md) for performance optimization
3. Inspect API responses for detailed error messages

**Last Updated:** February 2, 2026
