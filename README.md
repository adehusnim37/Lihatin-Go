# Lihatin-Go

A modern URL shortener REST API built with Go and Gin framework that provides comprehensive URL shortening, user management, authentication, and activity logging capabilities.

## 🚀 Features

- **URL Shortening**: Create, manage, and track short links with custom codes
- **Multi-Authentication System**: Email verification, TOTP (2FA), and secure login
- **User Management**: Complete CRUD operations with role-based access control (Admin/User)
- **Activity Logging**: Comprehensive logging with request tracking and analytics
- **Indonesian Validation**: User-friendly error messages in Bahasa Indonesia
- **Admin Dashboard**: Bulk operations and advanced link management
- **Link Analytics**: View tracking, click statistics, and performance metrics
- **Security Features**: Passcode protection, expiration dates, and access control
- **GORM + Raw SQL**: Hybrid database approach for optimal performance

## 🛠️ Tech Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin (v1.10.0)
- **Database**: MySQL with GORM ORM
- **Validation**: Custom Indonesian validation system with go-playground/validator (v10.26.0)
- **Authentication**: TOTP, Email verification, JWT-like tokens
- **Database Migration**: Auto-migration with GORM
- **Utilities**: UUID generation, password hashing, email sending

## 📁 Project Structure

```
├── main.go                    # Application entry point with auto-migration
├── go.mod                     # Go module dependencies
├── go.sum                     # Go module checksums
├── controllers/               # HTTP request handlers
│   ├── base.go               # Base controller with shared functionality
│   ├── auth/                 # Authentication controllers
│   │   ├── base.go           # Auth base controller
│   │   ├── login.go          # Login/logout endpoints
│   │   ├── email-auth.go     # Email verification system
│   │   └── totp.go           # TOTP (2FA) authentication
│   ├── shortlink/            # URL shortening controllers
│   │   ├── base.go           # Shortlink base controller
│   │   ├── create.go         # Create short links
│   │   ├── get.go            # Retrieve short links
│   │   ├── getall.go         # List all links (paginated)
│   │   ├── update.go         # Update short links
│   │   ├── delete.go         # Delete individual links
│   │   ├── bulkDelete.go     # Bulk delete operations (Admin)
│   │   ├── stats.go          # Link analytics and statistics
│   │   └── redirect.go       # Handle URL redirections
│   ├── user/                 # User management controllers
│   │   ├── base.go           # User controller base
│   │   ├── create.go         # User registration
│   │   ├── get.go            # User retrieval
│   │   ├── update.go         # User profile updates
│   │   └── delete.go         # User deletion
│   └── logger/               # Activity logging controllers
│       └── logger_controller.go # Logging endpoints
├── middleware/               # HTTP middleware
│   ├── activity_logger.go    # Activity logging middleware
│   ├── auth.go               # Authentication middleware
│   ├── admin_auth.go         # Admin role verification
│   └── loginAttempt.go       # Login attempt tracking
├── models/                   # Data models and structures
│   ├── common/               # Shared models
│   │   └── response.go       # Standard API response structure
│   ├── user/                 # User-related models
│   │   ├── user.go           # User data model
│   │   ├── auth.go           # Authentication models
│   │   └── api_key.go        # API key management
│   ├── shortlink/            # Short link models
│   │   ├── short_link.go     # Short link data model
│   │   ├── detail.go         # Link details and metadata
│   │   └── view.go           # View tracking model
│   ├── logging/              # Logging models
│   │   └── logger.go         # Activity log model
│   └── migrations/           # Database migrations
│       └── migration.go      # Migration utilities
├── repositories/             # Data access layer
│   ├── auth-repo.go          # Authentication repository
│   ├── user-repo.go          # User database operations
│   ├── user-auth-repo.go     # User authentication operations
│   ├── api-key-repo.go       # API key management
│   ├── login-attempt-repo.go # Login attempt tracking
│   ├── logger_repository.go  # Activity log operations
│   ├── auth-method-repo.go   # Auth method management
│   └── short-link/           # Short link repositories
│       └── short.go          # Short link database operations
├── routes/                   # API route definitions
│   ├── router.go             # Main router setup with middleware
│   ├── auth_routes.go        # Authentication routes
│   ├── user_routes.go        # User management routes
│   ├── short_routes.go       # Short link routes
│   └── logger_routes.go      # Activity log routes
├── dto/                      # Data Transfer Objects
│   └── short.go              # Short link request/response DTOs
├── utils/                    # Utility functions
│   ├── config.go             # Configuration management
│   ├── validator.go          # Indonesian validation system
│   ├── jwt.go                # JWT token utilities
│   ├── email.go              # Email sending utilities
│   ├── totp.go               # TOTP (2FA) utilities
│   ├── password.go           # Password hashing utilities
│   ├── csrf.go               # CSRF protection
│   ├── logger.go             # Custom logging utilities
│   ├── error.go              # Error handling utilities
│   ├── tokens.go             # Token management
│   ├── ip.go                 # IP address utilities
│   ├── parseua.go            # User agent parsing
│   ├── pagination.go         # Pagination utilities
│   └── misc.go               # Miscellaneous utilities
├── cmd/                      # Command line utilities
│   ├── migrate/              # Database migration commands
│   ├── test/                 # Testing utilities
│   └── test-email/           # Email testing tools
├── tests/                    # Test suites
│   ├── auth_system_verification.go # Auth system tests
│   └── system_verification.go      # System integration tests
└── examples/                 # Usage examples
    └── validation_example.go # Validation system examples
```

## 🚦 API Endpoints

### Short Links
- `POST /v1/short/create` - Create new short link
- `GET /v1/short/:code` - Get short link details
- `GET /v1/short/:code/stats` - Get link statistics and analytics
- `GET /v1/short/:code/views` - Get paginated view history
- `PUT /v1/short/:code` - Update short link (owner only)
- `DELETE /v1/short/:code` - Delete short link (with optional passcode)
- `GET /v1/short/redirect/:code` - Redirect to original URL (public)

### Short Links - User Operations
- `GET /v1/user/shorts` - Get user's short links (paginated)

### Short Links - Admin Operations
- `GET /v1/admin/shorts` - Get all short links (paginated, admin only)
- `DELETE /v1/admin/shorts/bulk-delete` - Bulk delete short links (admin only)
- `GET /v1/admin/shorts/:code` - Get any short link details (admin only)

### Authentication
- `POST /v1/auth/login` - User login with email/username and password
- `POST /v1/auth/logout` - User logout
- `POST /v1/auth/register` - User registration
- `POST /v1/auth/verify-email` - Email verification
- `POST /v1/auth/resend-verification` - Resend verification email
- `POST /v1/auth/forgot-password` - Password reset request
- `POST /v1/auth/reset-password` - Password reset confirmation

### Two-Factor Authentication (TOTP)
- `POST /v1/auth/totp/setup` - Setup TOTP (2FA)
- `POST /v1/auth/totp/verify` - Verify TOTP code
- `POST /v1/auth/totp/disable` - Disable TOTP
- `POST /v1/auth/totp/backup-codes` - Generate backup codes

### Users
- `GET /v1/users` - Get all users (admin only)
- `GET /v1/users/:id` - Get user by ID
- `POST /v1/users` - Create new user (admin only)
- `PUT /v1/users/:id` - Update user
- `DELETE /v1/users/:id` - Delete user (admin only)
- `GET /v1/users/profile` - Get current user profile
- `PUT /v1/users/profile` - Update current user profile

### Activity Logs
- `GET /v1/logs` - Get all activity logs (admin only)
- `GET /v1/logs/user/:username` - Get logs by username (admin only)

## 🔧 Installation & Setup

### Prerequisites
- Go 1.24.3 or higher
- MySQL 8.0 or higher
- Git

### Installation Steps

1. **Clone the repository**
   ```bash
   git clone https://github.com/adehusnim37/lihatin-go.git
   cd lihatin-go
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Setup Environment Variables**
   
   Create a `.env` file in the root directory:
   ```env
   # Database Configuration
   DATABASE_URL=username:password@tcp(localhost:3306)/lihatin_go?charset=utf8mb4&parseTime=True&loc=Local
   
   # Server Configuration
   APP_PORT=:8080
   
   # Email Configuration (for verification and notifications)
   SMTP_HOST=smtp.gmail.com
   SMTP_PORT=587
   SMTP_USER=your-email@gmail.com
   SMTP_PASS=your-app-password
   FROM_EMAIL=your-email@gmail.com
   
   # Security Keys
   JWT_SECRET=your-super-secret-jwt-key-here
   TOTP_ISSUER=Lihatin-Go
   ```

4. **Setup MySQL Database**
   ```sql
   CREATE DATABASE lihatin_go CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
   ```
   
   **Note**: The application uses GORM auto-migration, so all tables will be created automatically when you run the application for the first time.

5. **Run the application**
   ```bash
   go run main.go
   ```

   The server will start on `http://localhost:8080` (or the port specified in `APP_PORT`)

6. **Alternative: Using Air for development (hot reload)**
   ```bash
   # Install Air
   go install github.com/cosmtrek/air@latest
   
   # Run with hot reload
   air
   ```

## 🔒 Validation System

### Indonesian Error Messages
The application features a comprehensive validation system with user-friendly error messages in Bahasa Indonesia:

```json
{
  "success": false,
  "message": "Validasi gagal",
  "errors": {
    "original_url": "URL Asli wajib diisi",
    "custom_code": "Kode Kustom tidak boleh mengandung spasi"
  },
  "details": [
    {
      "field": "original_url",
      "message": "URL Asli wajib diisi"
    }
  ]
}
```

### Custom Validation Rules
- **`pwdcomplex`**: Password complexity (min 8 chars, uppercase, lowercase, number, symbol)
- **`username`**: Username format (letters, numbers, underscore, hyphen only)
- **`unique`**: Array uniqueness validation (no duplicates)
- **`no_space`**: No spaces allowed in field

### Short Link Validation
- **Original URL**: Required, valid URL format
- **Custom Code**: Optional, max 10 characters, alphanumeric + underscore/hyphen, no spaces
- **Passcode**: Optional, exactly 6 digits
- **Title**: Optional, max 255 characters
- **Description**: Optional, max 500 characters

### User Model Validation
- **Username**: Required, 3-30 characters, letters/numbers/underscore/hyphen only
- **Email**: Required, valid email format
- **Password**: Required, complex password with:
  - Minimum 8 characters
  - At least one uppercase letter
  - At least one lowercase letter
  - At least one number
  - At least one special symbol
- **Full Name**: Required, 2-100 characters

## 🛡️ Security Features

- **Multi-Factor Authentication**: TOTP (2FA) support with backup codes
- **Email Verification**: Required email verification for new accounts
- **Password Complexity**: Custom validation ensuring strong passwords
- **Role-Based Access Control**: Admin and User roles with different permissions
- **Passcode Protection**: Optional passcode protection for short links
- **Input Validation**: Comprehensive validation with Indonesian error messages
- **Activity Logging**: All actions logged with detailed information
- **Login Attempt Tracking**: Protection against brute force attacks
- **Token-Based Security**: Secure session management
- **CSRF Protection**: Cross-site request forgery protection

## 📊 Activity Logging

The application automatically logs comprehensive user activities including:
- **HTTP Details**: Method, route, status codes, and response times
- **Request Data**: Request bodies (with sensitive data redaction), query parameters, route parameters
- **User Context**: User agent, browser information, IP addresses, session data
- **Short Link Activities**: Link creation, updates, deletions, and access attempts
- **Authentication Events**: Login attempts, 2FA activities, email verifications
- **Performance Metrics**: Response time tracking for performance analysis
- **Security Events**: Failed login attempts, unauthorized access attempts

## 🔐 Authentication System

The application includes a comprehensive authentication system with:

### Basic Authentication
- **Password Hashing**: bcrypt encryption for all passwords
- **Flexible Login**: Support for both email and username login
- **Session Management**: Secure token-based authentication
- **Password Reset**: Email-based password recovery system

### Multi-Factor Authentication
- **TOTP (2FA)**: Time-based One-Time Password support
- **Backup Codes**: Recovery codes for 2FA emergencies
- **QR Code Generation**: Easy 2FA setup with QR codes
- **Device Management**: Per-user 2FA configuration

### Email Verification
- **Account Verification**: Required email verification for new accounts
- **Resend Verification**: Ability to resend verification emails
- **Secure Tokens**: Time-limited verification tokens

See detailed documentation:
- [AUTHENTICATION_SYSTEM_DOCS.md](AUTHENTICATION_SYSTEM_DOCS.md) - Complete auth system documentation
- [VALIDATION_IMPLEMENTATION.md](VALIDATION_IMPLEMENTATION.md) - Validation system details

## 🚀 Usage Examples

### Create a short link
```bash
curl -X POST http://localhost:8080/v1/short/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "original_url": "https://example.com/very-long-url",
    "title": "Example Website",
    "description": "A sample website for demonstration",
    "custom_code": "example123",
    "passcode": "123456"
  }'
```

### Get short link statistics
```bash
curl http://localhost:8080/v1/short/example123/stats \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Bulk delete short links (Admin only)
```bash
curl -X DELETE http://localhost:8080/v1/admin/shorts/bulk-delete \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ADMIN_TOKEN" \
  -d '{
    "codes": ["link1", "link2", "link3"]
  }'
```

### User registration
```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "SecurePass123!",
    "full_name": "John Doe"
  }'
```

### Login with credentials
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email_or_username": "john@example.com",
    "password": "SecurePass123!"
  }'
```

### Setup Two-Factor Authentication
```bash
curl -X POST http://localhost:8080/v1/auth/totp/setup \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "password": "SecurePass123!"
  }'
```

### Access short link (public - no auth required)
```bash
curl http://localhost:8080/v1/short/redirect/example123
```

### Validation Error Example
When validation fails, you'll receive Indonesian error messages:
```json
{
  "success": false,
  "message": "Validasi gagal",
  "errors": {
    "codes": "Kode Short Link tidak boleh ada yang duplikat",
    "original_url": "URL Asli wajib diisi"
  },
  "details": [
    {
      "field": "codes",
      "message": "Kode Short Link tidak boleh ada yang duplikat"
    },
    {
      "field": "original_url", 
      "message": "URL Asli wajib diisi"
    }
  ]
}
```

## 📚 Additional Documentation

- [AUTHENTICATION_SYSTEM_DOCS.md](AUTHENTICATION_SYSTEM_DOCS.md) - Complete authentication system documentation
- [VALIDATION_IMPLEMENTATION.md](VALIDATION_IMPLEMENTATION.md) - Indonesian validation system implementation
- [COMPLETION_SUMMARY.md](COMPLETION_SUMMARY.md) - Project completion and feature summary
- [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) - Production deployment guide
- [ENV_CONFIG_USAGE.md](ENV_CONFIG_USAGE.md) - Environment configuration documentation

## 🧪 Testing

The project includes comprehensive testing utilities:

```bash
# Run system verification tests
go run tests/system_verification.go

# Run authentication system tests
go run tests/auth_system_verification.go

# Test validation system examples
go run examples/validation_example.go

# Test email functionality
go run cmd/test-email/main.go

# Run database migrations manually
go run cmd/migrate/main.go
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the Unlicense - see the [LICENSE](LICENSE) file for details.

## 👨‍💻 Author

**Ade Husni Mubarrok** - [@adehusnim37](https://github.com/adehusnim37)

---

*Built with ❤️ using Go, Gin framework, and modern web technologies*
