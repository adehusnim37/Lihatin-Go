# Universal Controller Refactoring

## 🎯 Overview
Refactoring dua controller terpisah (`ListUserShortLinks` dan `ListAllShortLinks`) menjadi satu controller universal (`ListShortLinks`) dengan smart role-based filtering.

## ✅ What Changed

### 1. **Universal Controller Method**
- **File**: `controllers/short-link/getall.go`
- **New Method**: `ListShortLinks(ctx *gin.Context)`
- **Logic**: 
  - Auto-detects user role from JWT context
  - User role → filtered by `user_id` 
  - Admin role → no filter (all data)

### 2. **Smart Role-Based Filtering**
```go
if userRoleStr == "admin" {
    // Admin: Get all short links (no user filter)
    paginatedResponse, repositoryErr = c.repo.ListAllShortLinks(page, limit, sort, orderBy)
} else {
    // User: Get only user's short links (filtered by user_id)
    paginatedResponse, repositoryErr = c.repo.GetShortsByUserIDWithPagination(userIDStr, page, limit, sort, orderBy)
}
```

### 3. **Enhanced Admin Features**
- Admin mendapat opsi sort tambahan: `original_url`, `is_active`, `user_id`
- User hanya dapat: `created_at`, `updated_at`, `short_code`, `title`

### 4. **Updated Routes**
- **File**: `routes/short_routes.go`
- **User Route**: `/v1/users/me/shorts` → `shortController.ListShortLinks`
- **Admin Route**: `/v1/admin/shorts` → `shortController.ListShortLinks`
- **Same endpoint, different behavior based on role!**

### 5. **Backward Compatibility**
- **File**: `controllers/short-link/getall.go`
  - `ListUserShortLinks()` → wrapper untuk `ListShortLinks()`
- **File**: `controllers/short-link/getalladmin.go`  
  - `ListAllShortLinks()` → wrapper untuk `ListShortLinks()`

## 🚀 Usage Examples

### User Request (Filtered by user_id)
```bash
GET /v1/users/me/shorts?page=1&limit=10&sort=created_at&order_by=desc
Authorization: Bearer <user_jwt>

# Response: Only user's short links
```

### Admin Request (All data, no filter)
```bash
GET /v1/admin/shorts?page=1&limit=10&sort=user_id&order_by=asc
Authorization: Bearer <admin_jwt>

# Response: ALL short links from all users
```

## 🛡️ Security Features

1. **Automatic Role Detection**: From JWT claims in context
2. **Middleware Protection**: 
   - `AuthMiddleware` → validates JWT
   - `RequireRole("admin")` → admin-only endpoints
3. **Smart Filtering**: Users can ONLY see their own data
4. **Enhanced Logging**: Different log messages for user vs admin actions

## 📊 Benefits

1. **✅ Single Source of Truth**: Satu controller untuk semua list operations
2. **✅ Consistent Logic**: Same validation, pagination, error handling
3. **✅ Role-Based Features**: Admin gets more capabilities automatically  
4. **✅ Security**: Automatic filtering prevents data leakage
5. **✅ Maintainability**: Easier to update and debug
6. **✅ Backward Compatible**: Existing code tetap berfungsi

## 🔧 Technical Details

### Context Variables Used:
- `user_id`: From JWT claims (string)
- `role`: From JWT claims ("user" or "admin")

### Repository Methods:
- **User**: `GetShortsByUserIDWithPagination(userID, ...)`
- **Admin**: `ListAllShortLinks(...)`

### Response Types:
- **User**: `*dto.PaginatedShortLinksResponse`
- **Admin**: `*dto.PaginatedShortLinksAdminResponse`

## 🎉 Result

**Before**: 2 controllers, 2 endpoints, duplicate code, maintenance overhead
**After**: 1 universal controller, 2 endpoints, smart role detection, consistent behavior

Perfect refactoring! 🚀
