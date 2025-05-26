# Lihatin-Go

A modern REST API built with Go and Gin framework that provides comprehensive user management and activity logging capabilities.

## 🚀 Features

- **User Management**: Complete CRUD operations for user accounts
- **Enhanced Activity Logging**: Comprehensive logging with request bodies, parameters, and performance metrics
- **Premium User Protection**: Special handling for premium users
- **Input Validation**: Comprehensive validation with custom password complexity rules
- **MySQL Integration**: Robust database integration with connection pooling
- **RESTful API**: Clean, consistent API design following REST principles
- **Security Features**: Automatic sensitive data redaction and input sanitization
- **Performance Monitoring**: Response time tracking and analytics support

## 🛠️ Tech Stack

- **Language**: Go 1.24.3
- **Web Framework**: Gin (v1.10.0)
- **Database**: MySQL
- **Validation**: go-playground/validator (v10.26.0)
- **Database Driver**: go-sql-driver/mysql (v1.9.2)
- **UUID Generation**: google/uuid (v1.6.0)

## 📁 Project Structure

```
├── main.go                    # Application entry point
├── go.mod                     # Go module dependencies
├── go.sum                     # Go module checksums
├── controllers/               # HTTP request handlers
│   ├── base.go               # Base controller with shared functionality
│   ├── logger_controller.go  # Activity logging endpoints
│   └── user/                 # User-specific controllers
│       ├── base.go           # User controller base
│       ├── create.go         # User creation endpoint
│       ├── delete.go         # User deletion endpoint
│       ├── get.go            # User retrieval endpoints
│       └── update.go         # User update endpoint
├── middleware/               # HTTP middleware
│   └── activity_logger.go    # Activity logging middleware
├── models/                   # Data models and structures
│   ├── logger-model.go       # Activity log data model
│   ├── response.go           # API response structure
│   └── user-model.go         # User data model
├── repositories/             # Data access layer
│   ├── logger_repository.go  # Activity log database operations
│   └── user-repo.go          # User database operations
├── routes/                   # API route definitions
│   ├── logger_routes.go      # Activity log routes
│   ├── router.go             # Main router setup
│   └── user_routes.go        # User routes
└── utils/                    # Utility functions
    ├── csrf.go               # CSRF protection utilities
    └── validator.go          # Custom validation rules
```

## 🚦 API Endpoints

### Users
- `GET /v1/users` - Get all users
- `GET /v1/users/:id` - Get user by ID
- `POST /v1/users` - Create new user
- `PUT /v1/users/:id` - Update user
- `DELETE /v1/users/:id` - Delete user (with confirmation)

### Activity Logs
- `GET /v1/logs` - Get all activity logs
- `GET /v1/logs/user/:username` - Get logs by username

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

3. **Setup MySQL Database**
   ```sql
   CREATE DATABASE LihatinGo;
   USE LihatinGo;
   
   -- Create users table
   CREATE TABLE users (
       id VARCHAR(36) PRIMARY KEY,
       username VARCHAR(50) UNIQUE NOT NULL,
       first_name VARCHAR(50) NOT NULL,
       last_name VARCHAR(50) NOT NULL,
       email VARCHAR(100) UNIQUE NOT NULL,
       password VARCHAR(255) NOT NULL,
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
       updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
       deleted_at TIMESTAMP NULL,
       is_premium BOOLEAN DEFAULT FALSE,
       avatar VARCHAR(255) NOT NULL
   );
   
   -- Create activity logs table
   CREATE TABLE ActivityLog (
       id VARCHAR(36) PRIMARY KEY,
       level VARCHAR(50) NOT NULL,
       message VARCHAR(255) NOT NULL,
       username VARCHAR(50) NOT NULL,
       timestamp VARCHAR(50) NOT NULL,
       ipaddress VARCHAR(50) NOT NULL,
       useragent VARCHAR(255) NOT NULL,
       browserinfo VARCHAR(255) NOT NULL,
       action VARCHAR(100) NOT NULL,
       route VARCHAR(100) NOT NULL,
       method VARCHAR(10) NOT NULL,
       statuscode INT NOT NULL,
       requestbody TEXT,
       queryparams TEXT,
       routeparams TEXT,
       contextlocals TEXT,
       responsetime BIGINT,
       createdat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
       updatedat TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
       deletedat TIMESTAMP NULL
   );
   ```

4. **Configure Database Connection**
   
   Update the database connection string in `main.go`:
   ```go
   db, err := sql.Open("mysql", "username:password@tcp(localhost:3306)/LihatinGo")
   ```

5. **Run the application**
   ```bash
   go run main.go
   ```

   The server will start on `http://localhost:8880`

## 🔒 Validation Rules

### User Model Validation
- **Username**: Required, 3-50 characters
- **First Name**: Required, 3-50 characters  
- **Last Name**: Required, 3-50 characters
- **Email**: Required, valid email format
- **Password**: Required, 8-50 characters with complexity requirements:
  - At least one letter (a-z, A-Z)
  - At least one digit (0-9)
  - At least one special character (!@#$%^&*()_+-=[]{}|;':",./<>?)
- **Avatar**: Required, valid URL format

## 🛡️ Security Features

- **Password Complexity**: Custom validation ensuring strong passwords
- **Premium User Protection**: Premium users cannot be deleted
- **Username Confirmation**: Deletion requires username confirmation
- **Input Validation**: Comprehensive validation on all endpoints
- **Activity Logging**: All actions are logged with detailed information

## 📊 Activity Logging

The application automatically logs comprehensive user activities including:
- **HTTP Details**: Method, route, status codes, and response times
- **Request Data**: Request bodies (with sensitive data redaction), query parameters, route parameters
- **User Context**: User agent, browser information, IP addresses, session data
- **Performance Metrics**: Response time tracking for performance analysis
- **Security**: Automatic redaction of passwords, tokens, and other sensitive fields

See [ACTIVITY_LOGGER.md](ACTIVITY_LOGGER.md) for detailed documentation on the enhanced logging system.

## 🚀 Usage Examples

### Create a new user
```bash
curl -X POST http://localhost:8880/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "first_name": "John",
    "last_name": "Doe",
    "email": "john@example.com",
    "password": "SecurePass123!",
    "avatar": "https://example.com/avatar.jpg"
  }'
```

### Get all users
```bash
curl http://localhost:8880/v1/users
```

### Get activity logs
```bash
curl http://localhost:8880/v1/logs
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

**Ade Husni Mubaraq** - [@adehusnim37](https://github.com/adehusnim37)

---

*Built with ❤️ using Go and Gin framework*
