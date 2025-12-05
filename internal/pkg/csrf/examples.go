package csrf

// ===========================================================================
// USAGE EXAMPLES
// ===========================================================================

/*
=============================================================================
1. BASIC USAGE - Apply to all routes
=============================================================================

func SetupRouter() *gin.Engine {
    r := gin.Default()

    // Add CSRF middleware
    r.Use(csrf.Middleware(csrf.DefaultOptions()))

    // Your routes...
    r.POST("/api/transfer", handleTransfer)

    return r
}

=============================================================================
2. CUSTOM OPTIONS
=============================================================================

func SetupRouter() *gin.Engine {
    r := gin.Default()

    csrfOpts := csrf.Options{
        Secret:       []byte(os.Getenv("CSRF_SECRET")), // WAJIB 32+ bytes
        CookieName:   "_csrf_token",
        CookieDomain: ".yourdomain.com",
        MaxAge:       3600 * 24, // 24 hours
        Secure:       true,      // HTTPS only
        HttpOnly:     false,     // JS harus bisa baca untuk SPA
        SameSite:     http.SameSiteLaxMode,

        // Trusted origins untuk CORS
        TrustedOrigins: []string{
            "app.yourdomain.com",
            "admin.yourdomain.com",
        },

        // Skip CSRF untuk API key authenticated routes
        SkipPaths: []string{
            "/api/v1/webhook",
            "/api/v1/public",
        },

        // Custom error handler
        ErrorHandler: func(c *gin.Context) {
            c.JSON(403, gin.H{
                "error": "CSRF validation failed",
                "code": "CSRF_ERROR",
            })
            c.Abort()
        },
    }

    r.Use(csrf.Middleware(csrfOpts))
    return r
}

=============================================================================
3. GET CSRF TOKEN ENDPOINT (untuk SPA)
=============================================================================

// GET /api/v1/csrf-token
func GetCSRFToken(c *gin.Context) {
    // Token sudah di-generate oleh middleware
    token := csrf.GetMaskedToken(c) // Masked untuk security

    c.JSON(200, gin.H{
        "csrfToken": token,
    })
}

=============================================================================
4. FRONTEND IMPLEMENTATION (JavaScript/React)
=============================================================================

// Step 1: Get token saat app load
const response = await fetch('/api/v1/csrf-token', {
    credentials: 'include' // PENTING: untuk kirim cookie
});
const data = await response.json();
const csrfToken = data.csrfToken;

// Step 2: Store token (bisa di memory atau localStorage)
localStorage.setItem('csrfToken', csrfToken);

// Step 3: Kirim di setiap POST/PUT/DELETE request
const result = await fetch('/api/v1/transfer', {
    method: 'POST',
    credentials: 'include',
    headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': localStorage.getItem('csrfToken')
    },
    body: JSON.stringify({ to: 'user123', amount: 100 })
});

// Atau dengan Axios (recommended)
import axios from 'axios';

const api = axios.create({
    baseURL: 'https://api.yourdomain.com',
    withCredentials: true
});

// Interceptor untuk auto-attach token
api.interceptors.request.use(config => {
    const token = localStorage.getItem('csrfToken');
    if (token && ['post', 'put', 'delete', 'patch'].includes(config.method)) {
        config.headers['X-CSRF-Token'] = token;
    }
    return config;
});

=============================================================================
5. SELECTIVE CSRF (hanya untuk authenticated routes)
=============================================================================

func SetupRouter() *gin.Engine {
    r := gin.Default()

    // Public routes - no CSRF
    public := r.Group("/public")
    {
        public.GET("/health", healthCheck)
    }

    // Protected routes - with CSRF
    protected := r.Group("/api/v1")
    protected.Use(AuthMiddleware())
    protected.Use(csrf.Middleware(csrf.DefaultOptions()))
    {
        protected.POST("/transfer", handleTransfer)
    }

    return r
}

=============================================================================
6. ENVIRONMENT VARIABLES
=============================================================================

# .env file
CSRF_SECRET=your-32-bytes-or-more-secret-key-here
ENV=production

# Generate secret:
# openssl rand -hex 32

*/
