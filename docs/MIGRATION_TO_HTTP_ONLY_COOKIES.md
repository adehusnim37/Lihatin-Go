# 🔄 Migration Guide: localStorage to HTTP-Only Cookie Authentication

## Purpose
This guide helps you migrate existing code that uses localStorage token management to the new HTTP-Only cookie-based authentication system.

## 🚨 Breaking Changes

### 1. Token Storage
**Before**:
```typescript
// Tokens stored in localStorage
localStorage.setItem('access_token', token);
localStorage.setItem('refresh_token', refreshToken);
```

**After**:
```typescript
// Tokens stored in HTTP-Only cookies automatically by backend
// No manual token storage needed!
```

### 2. API Requests
**Before**:
```typescript
fetch('/api/endpoint', {
  method: 'GET',
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
});
```

**After**:
```typescript
import { fetchWithAuth } from '@/lib/api/fetch-wrapper';

// Cookies sent automatically, token refresh handled automatically
fetchWithAuth('/v1/endpoint', {
  method: 'GET'
});
```

### 3. Authentication Check
**Before**:
```typescript
const isAuthenticated = !!localStorage.getItem('access_token');
```

**After**:
```typescript
import { checkAuth } from '@/lib/api/auth';

const isAuthenticated = await checkAuth(); // Validates with backend
```

### 4. Login Flow
**Before**:
```typescript
const response = await login(credentials);
if (response.success) {
  saveTokens(response.data.token, response.data.refresh_token);
  router.push('/dashboard');
}
```

**After**:
```typescript
import { useAuth } from '@/app/context/AuthContext';

const auth = useAuth();
const response = await login(credentials);
if (response.success) {
  saveUserData(response.data.user); // Save profile only
  await auth.login(); // Update auth context
  router.push('/main');
}
```

### 5. Logout Flow
**Before**:
```typescript
localStorage.removeItem('access_token');
localStorage.removeItem('refresh_token');
router.push('/auth/login');
```

**After**:
```typescript
import { useAuth } from '@/app/context/AuthContext';

const auth = useAuth();
await auth.logout(); // Calls backend, clears cookies, redirects
```

## 📝 Step-by-Step Migration

### Step 1: Update API Calls

Replace all API calls that manually set Authorization headers:

```typescript
// ❌ OLD CODE
const fetchUserProfile = async () => {
  const token = localStorage.getItem('access_token');
  const response = await fetch('/v1/auth/profile', {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });
  return response.json();
};

// ✅ NEW CODE
import { getWithAuth } from '@/lib/api/fetch-wrapper';

const fetchUserProfile = async () => {
  const response = await getWithAuth('/v1/auth/profile');
  return response.json();
};
```

### Step 2: Remove Token Management

Delete all token storage and retrieval code:

```typescript
// ❌ DELETE THESE FUNCTIONS
function saveTokens(accessToken, refreshToken) { ... }
function getAccessToken() { ... }
function getRefreshToken() { ... }
function clearTokens() { ... }
```

### Step 3: Use AuthContext

Replace custom auth checks with AuthContext:

```typescript
// ❌ OLD CODE
const [isAuthenticated, setIsAuthenticated] = useState(false);

useEffect(() => {
  const token = localStorage.getItem('access_token');
  setIsAuthenticated(!!token);
}, []);

// ✅ NEW CODE
import { useAuth } from '@/app/context/AuthContext';

const { isAuthenticated, isLoading } = useAuth();
```

### Step 4: Update Protected Components

Replace localStorage checks with AuthContext:

```typescript
// ❌ OLD CODE
function ProtectedComponent() {
  const token = localStorage.getItem('access_token');
  
  if (!token) {
    return <Redirect to="/auth/login" />;
  }
  
  return <div>Protected Content</div>;
}

// ✅ NEW CODE
import { useAuth } from '@/app/context/AuthContext';

function ProtectedComponent() {
  const { isAuthenticated, isLoading } = useAuth();
  
  if (isLoading) {
    return <Loading />;
  }
  
  if (!isAuthenticated) {
    return <Redirect to="/auth/login" />;
  }
  
  return <div>Protected Content</div>;
}
```

### Step 5: Update Custom Fetch Wrappers

If you have custom fetch wrappers, replace them with the new `fetchWithAuth`:

```typescript
// ❌ OLD CODE
async function authenticatedFetch(url, options = {}) {
  const token = localStorage.getItem('access_token');
  const response = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
      'Authorization': `Bearer ${token}`
    }
  });
  
  if (response.status === 401) {
    // Manual token refresh logic...
  }
  
  return response;
}

// ✅ NEW CODE - JUST USE fetchWithAuth
import { fetchWithAuth } from '@/lib/api/fetch-wrapper';

// Automatic token refresh on 401!
const response = await fetchWithAuth(url, options);
```

## 🔍 Common Migration Issues

### Issue 1: "No token in response"
**Symptom**: Frontend expects `token` field in login/register response

**Solution**: Tokens are now in cookies, not response body
```typescript
// ❌ OLD CODE
const { token, refresh_token } = response.data;
saveTokens(token, refresh_token);

// ✅ NEW CODE
// No token in response! Cookies set automatically
const { user, auth } = response.data;
saveUserData(user);
```

### Issue 2: "401 Unauthorized on every request"
**Symptom**: Backend doesn't receive cookies

**Solution**: Add `credentials: 'include'` to all fetch calls
```typescript
// ❌ MISSING credentials
fetch('/v1/auth/profile', { method: 'GET' });

// ✅ WITH credentials
fetch('/v1/auth/profile', { 
  method: 'GET',
  credentials: 'include' // Required for cookies!
});
```

### Issue 3: "CORS error with cookies"
**Symptom**: Browser blocks cookies in cross-origin requests

**Solution**: Backend must have proper CORS configuration
```go
// Backend CORS middleware must include:
c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
c.Writer.Header().Set("Access-Control-Allow-Origin", origin) // Specific origin, NOT "*"
```

### Issue 4: "Cookies not set on login"
**Symptom**: Login succeeds but no cookies in browser

**Checklist**:
1. Backend sets cookies with `ctx.SetCookie()` ✓
2. Cookie path is `/` or matches request path ✓
3. Cookie domain matches current domain ✓
4. If HTTPS, `Secure` flag is set ✓
5. CORS allows credentials ✓
6. Frontend uses `credentials: 'include'` ✓

### Issue 5: "Infinite redirect loop"
**Symptom**: Page keeps redirecting between /login and /main

**Solution**: Check middleware and AuthContext logic
```typescript
// Ensure Next.js middleware checks cookie correctly
const accessToken = request.cookies.get('access_token')?.value;
if (isProtectedRoute && !accessToken) {
  // Redirect to login
}
```

## ✅ Migration Checklist

Use this checklist to ensure complete migration:

### Backend
- [ ] Login endpoint sets HTTP-Only cookies
- [ ] Refresh token endpoint reads from cookies
- [ ] Logout endpoint clears cookies
- [ ] AuthMiddleware supports cookie-based auth
- [ ] CORS configured with credentials: true
- [ ] /auth/me endpoint created for auth checks

### Frontend
- [ ] All `localStorage.setItem('access_token', ...)` removed
- [ ] All `localStorage.getItem('access_token')` removed
- [ ] All `Authorization: Bearer ${token}` headers removed
- [ ] All fetch calls use `credentials: 'include'`
- [ ] Login flow uses AuthContext.login()
- [ ] Logout flow uses AuthContext.logout()
- [ ] Protected routes use AuthContext or Next.js middleware
- [ ] Token refresh handled by fetchWithAuth wrapper

### Testing
- [ ] Login sets cookies in browser
- [ ] Protected routes redirect to login without cookie
- [ ] API calls include cookies automatically
- [ ] Token refresh works on 401 errors
- [ ] Logout clears cookies
- [ ] /auth/me returns user info with valid cookie
- [ ] CORS allows credentials from frontend origin

## 🎓 Learning Resources

### Understanding HTTP-Only Cookies
- **What**: Cookies that cannot be accessed by JavaScript
- **Why**: Prevents XSS attacks from stealing tokens
- **How**: Backend sets `httpOnly: true` flag on cookies

### Understanding CORS with Credentials
- **What**: Cross-Origin Resource Sharing with cookies
- **Why**: Allows cookies in cross-origin requests
- **How**: Backend sets `Access-Control-Allow-Credentials: true`

### Understanding Token Rotation
- **What**: Replacing refresh tokens on each use
- **Why**: Minimizes risk of token reuse attacks
- **How**: Backend deletes old token, creates new one

## 📞 Need Help?

If you encounter issues not covered here:

1. **Check Browser DevTools**
   - Network tab: Verify `Set-Cookie` headers
   - Application tab: Check cookies are stored
   - Console: Look for CORS or auth errors

2. **Check Backend Logs**
   - Look for "Token read from HTTP-Only cookie"
   - Verify cookie parsing in AuthMiddleware
   - Check CORS origin matching

3. **Review Documentation**
   - `HTTP_ONLY_COOKIE_AUTH_IMPLEMENTATION.md` - Full implementation details
   - `AUTHENTICATION_SYSTEM_DOCS.md` - Overall auth system
   - `JWT_IMPLEMENTATION_SUMMARY.md` - JWT details

4. **Common Solutions**
   - Clear browser cookies and localStorage
   - Restart backend server
   - Verify environment variables
   - Check CORS configuration matches frontend URL

## 🚀 Post-Migration Benefits

After migration, you'll have:

1. **Better Security**: XSS attacks cannot steal tokens
2. **Simpler Code**: No manual token management
3. **Auto Refresh**: Tokens refresh automatically on 401
4. **Type Safety**: Full TypeScript support
5. **Consistency**: Same pattern across all API calls
6. **Production Ready**: Follows security best practices

## 📈 Next Steps

After completing migration:

1. **Test Thoroughly**: Run through all auth flows
2. **Update Documentation**: Document your API usage patterns
3. **Monitor Logs**: Watch for auth-related errors
4. **Deploy Gradually**: Use feature flags if possible
5. **Educate Team**: Share this guide with team members

---

**Last Updated**: December 2024
**Version**: 1.0.0
