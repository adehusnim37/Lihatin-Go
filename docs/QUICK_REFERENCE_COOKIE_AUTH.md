# 🚀 Quick Reference: HTTP-Only Cookie Authentication

## TL;DR

**Tokens are now in HTTP-Only cookies, NOT localStorage!**

## 🔥 Quick Start

### Making Authenticated API Calls

```typescript
import { fetchWithAuth } from '@/lib/api/fetch-wrapper';

// GET request
const response = await fetchWithAuth('/v1/auth/profile', {
  method: 'GET'
});

// POST request
const response = await fetchWithAuth('/v1/auth/update-profile', {
  method: 'POST',
  body: JSON.stringify({ name: 'John' })
});

// Tokens sent automatically via cookies!
// Auto-refreshes on 401 errors!
```

### Using Authentication Context

```typescript
import { useAuth } from '@/app/context/AuthContext';

function MyComponent() {
  const { isAuthenticated, isLoading, logout } = useAuth();
  
  if (isLoading) return <Loading />;
  if (!isAuthenticated) return <Redirect to="/auth/login" />;
  
  return (
    <div>
      <h1>Protected Content</h1>
      <button onClick={logout}>Logout</button>
    </div>
  );
}
```

### Login Flow

```typescript
import { login, saveUserData } from '@/lib/api/auth';
import { useAuth } from '@/app/context/AuthContext';

const auth = useAuth();

const handleLogin = async (credentials) => {
  const response = await login(credentials);
  if (response.success) {
    saveUserData(response.data.user); // Save profile to localStorage
    await auth.login(); // Update auth context
    router.push('/main');
  }
};
```

### Logout Flow

```typescript
import { useAuth } from '@/app/context/AuthContext';

const auth = useAuth();

const handleLogout = async () => {
  await auth.logout(); // Calls backend, clears cookies, redirects
};
```

## 📋 Common Patterns

### Checking Authentication

```typescript
// In a component
const { isAuthenticated } = useAuth();

// In a function (async)
import { checkAuth } from '@/lib/api/auth';
const isAuth = await checkAuth();
```

### Protected Route

```typescript
// Next.js middleware handles this automatically!
// Just add route to protectedRoutes array in middleware.ts

// Or check in component:
const { isAuthenticated, isLoading } = useAuth();

if (isLoading) return <Spinner />;
if (!isAuthenticated) return <Navigate to="/auth/login" />;
```

### Manual Token Refresh

```typescript
import { refreshToken } from '@/lib/api/auth';

try {
  await refreshToken(); // New cookies set automatically
  console.log('Token refreshed!');
} catch (error) {
  console.error('Refresh failed:', error);
  // Redirect to login
}
```

## ⚠️ Don'ts

```typescript
// ❌ DON'T access tokens directly
localStorage.getItem('access_token'); // Won't work!

// ❌ DON'T set Authorization headers manually
headers: { 'Authorization': `Bearer ${token}` }; // Not needed!

// ❌ DON'T forget credentials: 'include'
fetch('/api/endpoint'); // Missing credentials!

// ❌ DON'T expect tokens in response
const { token } = response.data; // Tokens are in cookies!
```

## ✅ Do's

```typescript
// ✅ DO use fetchWithAuth for API calls
import { fetchWithAuth } from '@/lib/api/fetch-wrapper';
await fetchWithAuth('/v1/endpoint');

// ✅ DO use credentials: 'include' in fetch
fetch('/api/endpoint', { credentials: 'include' });

// ✅ DO use AuthContext for auth state
const { isAuthenticated } = useAuth();

// ✅ DO save user profile (NOT tokens)
saveUserData(response.data.user);
```

## 🔧 API Endpoints

### Public Endpoints (No Auth)
```
POST /v1/auth/login           - Login user
POST /v1/auth/register        - Register new user
POST /v1/auth/forgot-password - Request password reset
GET  /v1/auth/verify-email    - Verify email with token
```

### Protected Endpoints (Auth Required)
```
GET  /v1/auth/me              - Get current user (for auth check)
POST /v1/auth/logout          - Logout (clears cookies)
POST /v1/auth/refresh-token   - Refresh tokens (reads from cookie)
GET  /v1/auth/profile         - Get user profile
PUT  /v1/auth/profile         - Update user profile
```

## 🍪 Cookie Details

### Cookie Names
- `access_token` - Short-lived JWT (48 hours)
- `refresh_token` - Long-lived token (168 hours = 7 days)

### Cookie Flags
- `HttpOnly: true` - Not accessible to JavaScript
- `Secure: true` - Only sent over HTTPS (production)
- `Path: /` - Available for all routes
- `SameSite: Lax` - Reasonable CSRF protection

## 🔍 Debugging

### Check if cookies are set
```typescript
// Browser DevTools > Application > Cookies
// Look for: access_token, refresh_token
```

### Check if cookies are sent
```typescript
// Browser DevTools > Network > Select request > Headers
// Look for: Cookie: access_token=...; refresh_token=...
```

### Check backend response
```typescript
// Browser DevTools > Network > Select response > Headers
// Look for: Set-Cookie: access_token=...; HttpOnly; Secure
```

### Common Issues

**Issue**: "401 Unauthorized"
**Fix**: Check if cookies are being sent (credentials: 'include')

**Issue**: "No cookies in browser"
**Fix**: Check backend sets cookies correctly, CORS allows credentials

**Issue**: "Infinite redirect"
**Fix**: Check middleware cookie detection logic

## 📦 Helper Functions

### Available in `lib/api/auth.ts`
```typescript
login(credentials)           // Login user
register(userData)           // Register user
forgotPassword(request)      // Request password reset
refreshToken()               // Refresh tokens
logout()                     // Logout user
checkAuth()                  // Check if authenticated
saveUserData(user)           // Save profile to localStorage
getUserData()                // Get profile from localStorage
clearUserData()              // Clear profile from localStorage
```

### Available in `lib/api/fetch-wrapper.ts`
```typescript
fetchWithAuth(url, options)  // Fetch with auto-refresh
getWithAuth(url)             // GET request
postWithAuth(url, body)      // POST request
putWithAuth(url, body)       // PUT request
deleteWithAuth(url)          // DELETE request
```

## 🎯 Key Takeaways

1. **Tokens are in cookies** - No localStorage access needed
2. **Use `credentials: 'include'`** - Required for cookie transmission
3. **Use `fetchWithAuth`** - Automatic token refresh on 401
4. **Use `AuthContext`** - Centralized auth state management
5. **Backend validates cookies** - Middleware reads from cookies automatically

## 📚 Full Documentation

For detailed information, see:
- `HTTP_ONLY_COOKIE_AUTH_IMPLEMENTATION.md` - Complete implementation guide
- `MIGRATION_TO_HTTP_ONLY_COOKIES.md` - Migration from localStorage
- `AUTHENTICATION_SYSTEM_DOCS.md` - Overall authentication system

---

**Need Help?** Check the migration guide or full implementation docs!
