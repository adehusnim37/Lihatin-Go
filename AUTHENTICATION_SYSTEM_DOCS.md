# Authentication System Documentation

## Overview

This document describes the comprehensive authentication system implemented for the Lihatin-Go application. The system includes JWT-based authentication, TOTP (Time-based One-Time Password) for 2FA, email verification, admin user management, API key management, and detailed security monitoring.

## Features

### 1. Core Authentication
- **User Registration**: Email and username-based registration with password validation
- **User Login**: Secure login with password hashing and JWT token generation
- **Password Management**: Secure password reset via email verification
- **JWT Tokens**: Access and refresh token system for session management
- **Account Lockout**: Automatic account locking after failed login attempts

### 2. Multi-Factor Authentication (MFA)
- **TOTP Support**: Time-based One-Time Password using apps like Google Authenticator
- **Setup Process**: QR code generation for easy TOTP app configuration
- **Recovery Codes**: Backup codes for account recovery when TOTP device is unavailable
- **Disable/Enable**: Users can manage their 2FA settings

### 3. Email Verification
- **Registration Verification**: Email verification required for new accounts
- **Re-send Verification**: Users can request new verification emails
- **Password Reset**: Secure password reset via email tokens
- **Token Expiration**: Time-limited tokens for security

### 4. Admin User Management
- **Role-Based Access Control**: `user`, `admin`, `super_admin` roles
- **User Account Locking**: Admins can lock/unlock user accounts with reasons
- **User Listing**: Paginated user management interface
- **Login Monitoring**: View login attempts and security events

### 5. API Key Management
- **Key Generation**: Secure API key creation with customizable permissions
- **Key Management**: Update, revoke, and list API keys
- **Permission System**: Granular permissions (`read`, `write`, `admin`)
- **Expiration Dates**: Optional expiration for API keys
- **Usage Tracking**: Last used timestamps for monitoring

### 6. Security Monitoring
- **Login Attempts**: Detailed logging of all login attempts (successful and failed)
- **Activity Logging**: Comprehensive request/response logging
- **IP Tracking**: IP address logging for security analysis
- **User Agent Tracking**: Device and browser information logging

## Database Schema

### Users Table (Enhanced)
```sql
-- New admin fields added to existing users table
ALTER TABLE users 
ADD COLUMN is_locked BOOLEAN DEFAULT FALSE,
ADD COLUMN locked_at TIMESTAMP NULL,
ADD COLUMN locked_reason VARCHAR(500),
ADD COLUMN role ENUM('user', 'admin', 'super_admin') DEFAULT 'user';
```

### UserAuth Table
```sql
CREATE TABLE UserAuth (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    is_email_verified BOOLEAN DEFAULT FALSE,
    email_verification_token VARCHAR(255),
    email_verification_token_expires_at TIMESTAMP NULL,
    password_reset_token VARCHAR(255),
    password_reset_token_expires_at TIMESTAMP NULL,
    last_login_at TIMESTAMP NULL,
    failed_login_attempts INT DEFAULT 0,
    lockout_until TIMESTAMP NULL,
    is_active BOOLEAN DEFAULT TRUE,
    is_totp_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### APIKeys Table
```sql
CREATE TABLE APIKeys (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    last_used_at TIMESTAMP NULL,
    expires_at TIMESTAMP NULL,
    is_active BOOLEAN DEFAULT TRUE,
    permissions JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### LoginAttempts Table
```sql
CREATE TABLE LoginAttempts (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36),
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    failure_reason VARCHAR(100),
    attempted_email_or_username VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);
```

### AuthMethod Table
```sql
CREATE TABLE AuthMethod (
    id VARCHAR(36) PRIMARY KEY,
    user_auth_id VARCHAR(36) NOT NULL,
    type ENUM('totp', 'email_otp', 'magic_link', 'oauth_google', 'oauth_github', 'fido2') NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP NULL,
    last_used_at TIMESTAMP NULL,
    friendly_name VARCHAR(100),
    secret TEXT,
    recovery_codes JSON,
    provider_user_id VARCHAR(255),
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (user_auth_id) REFERENCES UserAuth(id) ON DELETE CASCADE
);
```

## API Endpoints

### Public Authentication Endpoints
```
POST /api/auth/login                    # User login
POST /api/auth/register                 # User registration
POST /api/auth/forgot-password          # Request password reset
POST /api/auth/reset-password           # Reset password with token
GET  /api/auth/verify-email             # Verify email with token
POST /api/auth/resend-verification      # Resend verification email
POST /api/auth/refresh-token            # Refresh JWT tokens
```

### Protected User Endpoints (Requires Auth)
```
POST /api/auth/logout                   # Logout user
POST /api/auth/change-password          # Change password
GET  /api/auth/profile                  # Get user profile
POST /api/auth/send-verification-email  # Send verification email

# TOTP Management
POST /api/auth/totp/setup               # Setup TOTP 2FA
POST /api/auth/totp/verify              # Verify TOTP code
POST /api/auth/totp/disable             # Disable TOTP 2FA
GET  /api/auth/totp/recovery-codes      # Get recovery codes
POST /api/auth/totp/regenerate-recovery-codes # Regenerate recovery codes
```

### API Key Management (Requires Auth)
```
GET    /api/api-keys/        # List user's API keys
POST   /api/api-keys/        # Create new API key
PUT    /api/api-keys/:id     # Update API key
DELETE /api/api-keys/:id     # Revoke API key
```

### Admin Endpoints (Requires Admin Role)
```
GET  /api/auth/admin/users              # List all users (paginated)
POST /api/auth/admin/users/:id/lock     # Lock user account
POST /api/auth/admin/users/:id/unlock   # Unlock user account
GET  /api/auth/admin/login-attempts     # View login attempts (paginated)
```

### Super Admin Endpoints (Requires Super Admin Role)
```
# Additional endpoints can be added here for super admin functionality
```

## Middleware

### 1. Authentication Middleware
- **Purpose**: Validates JWT tokens and extracts user information
- **Usage**: Applied to all protected routes
- **File**: `middleware/auth.go`

### 2. Admin Authentication Middleware
- **Purpose**: Ensures user has admin or super_admin role
- **Usage**: Applied to admin-only routes
- **File**: `middleware/admin_auth.go`

### 3. Activity Logger Middleware
- **Purpose**: Logs all HTTP requests and responses for monitoring
- **Usage**: Applied globally to all routes
- **File**: `middleware/activity_logger.go`

### 4. Email Verification Middleware
- **Purpose**: Ensures user has verified their email address
- **Usage**: Applied to routes requiring email verification
- **File**: `middleware/auth.go`

## Security Features

### 1. Password Security
- **Hashing**: Bcrypt with configurable cost
- **Validation**: Strong password requirements
- **Reset**: Secure token-based password reset

### 2. JWT Security
- **Signing**: HMAC SHA256 with secret key
- **Expiration**: Configurable token lifetime
- **Refresh**: Separate refresh token system
- **Claims**: User ID, email, role, and permissions

### 3. Rate Limiting
- **Login Attempts**: Account lockout after failed attempts
- **Token Refresh**: Prevent token refresh abuse
- **API Requests**: Rate limiting per user/IP

### 4. Data Protection
- **Sensitive Data**: Never expose passwords, tokens, or API keys in responses
- **API Key Preview**: Only show first 8 characters of API keys
- **Token Expiration**: Time-limited tokens for all operations

## Configuration

### Environment Variables
```bash
# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=username
DB_PASSWORD=password
DB_NAME=LihatinGo

# JWT Configuration
JWT_SECRET=your-secret-key
JWT_EXPIRATION=24h
REFRESH_TOKEN_EXPIRATION=168h

# Email Configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
EMAIL_FROM=noreply@yourdomain.com

# Application
PORT=8080
ENVIRONMENT=production
```

## Usage Examples

### 1. User Registration
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "SecurePassword123!"
  }'
```

### 2. User Login
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "SecurePassword123!"
  }'
```

### 3. Setup TOTP
```bash
curl -X POST http://localhost:8080/api/auth/totp/setup \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json"
```

### 4. Create API Key
```bash
curl -X POST http://localhost:8080/api/api-keys/ \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My API Key",
    "permissions": ["read", "write"],
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

### 5. Admin User Listing
```bash
curl -X GET "http://localhost:8080/api/auth/admin/users?page=1&limit=10" \
  -H "Authorization: Bearer YOUR_ADMIN_JWT_TOKEN"
```

### 6. Lock User Account (Admin)
```bash
curl -X POST http://localhost:8080/api/auth/admin/users/user-id/lock \
  -H "Authorization: Bearer YOUR_ADMIN_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Suspicious activity detected"
  }'
```

## File Structure

```
/controllers/auth/
├── base.go           # Main authentication controller
├── login.go          # Login-specific handlers
├── totp.go           # TOTP/2FA handlers
└── email-auth.go     # Email verification handlers

/middleware/
├── auth.go           # JWT authentication middleware
├── admin_auth.go     # Admin role middleware
└── activity_logger.go # Request logging middleware

/models/
├── auth-model.go     # Authentication request/response models
├── user-model.go     # User model with admin fields
├── user-auth-model.go # User authentication model
├── api-key-model.go  # API key models
└── logger-model.go   # Activity logging models

/repositories/
├── auth-repo.go      # Main authentication repository
├── user-repo.go      # User repository with admin methods
├── user-auth-repo.go # User auth repository
├── api-key-repo.go   # API key repository
└── login-attempt-repo.go # Login attempt repository

/utils/
├── jwt.go            # JWT token utilities
├── password.go       # Password hashing utilities
├── totp.go           # TOTP utilities
├── email.go          # Email sending utilities
└── tokens.go         # Token generation utilities

/migrations/
├── 001_add_enhanced_logging_fields.sql
├── 002_create_auth_tables.sql
└── 003_add_admin_fields_to_users.sql
```

## Testing

### Run Migrations
```bash
go run cmd/migrate/main.go
```

### Build Application
```bash
go build
```

### Run Application
```bash
./lihatin-go
```

### Test Endpoints
Use the provided curl examples or tools like Postman to test the API endpoints.

## Deployment Considerations

### 1. Security
- Use HTTPS in production
- Secure JWT secret keys
- Implement proper CORS policies
- Use secure email configurations

### 2. Database
- Regular backups
- Index optimization for performance
- Connection pooling
- Monitoring and alerting

### 3. Monitoring
- Log aggregation and analysis
- Performance monitoring
- Security event alerting
- User activity analytics

### 4. Scaling
- Load balancing for multiple instances
- Session storage considerations
- Database read replicas
- Caching strategies

## Future Enhancements

1. **OAuth Integration**: Google, GitHub, Facebook login
2. **FIDO2/WebAuthn**: Passwordless authentication
3. **Magic Links**: Email-based passwordless login
4. **Advanced Rate Limiting**: Redis-based distributed rate limiting
5. **Audit Logging**: Comprehensive audit trail for compliance
6. **User Sessions**: Active session management and termination
7. **Device Management**: Track and manage user devices
8. **Geographic Restrictions**: Location-based access controls

## Support and Maintenance

### Logs Location
- Application logs: Check server console output
- Database logs: MySQL/PostgreSQL logs
- Activity logs: Stored in ActivityLog table

### Common Issues
1. **Token Expiration**: Check JWT expiration settings
2. **Email Delivery**: Verify SMTP configuration
3. **Database Connection**: Check database credentials and connectivity
4. **Permission Errors**: Verify user roles and middleware configuration

### Backup Strategy
1. Regular database backups
2. Configuration file backups
3. User uploaded content backups
4. Log file rotation and archival

---

This authentication system provides enterprise-grade security features suitable for production applications. Regular security audits and updates are recommended to maintain optimal security posture.
