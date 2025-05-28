# Lihatin-Go Production Deployment Guide

## Quick Production Setup

### Prerequisites
- Go 1.24.3 or later
- MySQL or PostgreSQL database
- SMTP server for email services (optional)

### 1. Environment Configuration

Create a `.env` file in the root directory:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=your_db_user
DB_PASSWORD=your_db_password
DB_NAME=lihatin_go

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
JWT_EXPIRY=24h
REFRESH_TOKEN_EXPIRY=168h

# Email Configuration (Optional)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your_email@gmail.com
SMTP_PASSWORD=your_app_password
FROM_EMAIL=noreply@yourapp.com

# Application Configuration
PORT=8080
GIN_MODE=release
BCRYPT_COST=12

# CORS Configuration
ALLOWED_ORIGINS=http://localhost:3000,https://yourapp.com
ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
ALLOWED_HEADERS=Content-Type,Authorization
```

### 2. Database Setup

1. **Create Database**:
   ```sql
   CREATE DATABASE lihatin_go CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
   ```

2. **Apply Migrations**:
   ```bash
   go run cmd/migrate/main.go
   ```

### 3. Build and Deploy

1. **Build the Application**:
   ```bash
   go build -o lihatin-go
   ```

2. **Run the Application**:
   ```bash
   ./lihatin-go
   ```

3. **Verify Deployment**:
   ```bash
   go run cmd/test/main.go -url http://your-server:8080
   ```

### 4. Create Initial Admin User

After deployment, create an initial admin user:

1. **Register a normal user** via API or web interface
2. **Update the user role** in the database:
   ```sql
   UPDATE users SET role = 'super_admin' WHERE email = 'admin@yourcompany.com';
   ```

### 5. Production Considerations

#### Security
- Use strong JWT secrets (at least 32 characters)
- Enable HTTPS in production
- Configure proper CORS settings
- Set up rate limiting (consider nginx rate limiting)
- Regular security audits

#### Performance
- Configure database connection pooling
- Set up caching for frequently accessed data
- Monitor application performance
- Set up database indexing

#### Monitoring
- Set up application logging
- Monitor database performance
- Set up alerts for failed login attempts
- Monitor API response times

#### Backup
- Regular database backups
- Configuration file backups
- Test restore procedures

### 6. Docker Deployment (Optional)

Create a `Dockerfile`:

```dockerfile
FROM golang:1.24.3-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o lihatin-go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/lihatin-go .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080
CMD ["./lihatin-go"]
```

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=db
      - DB_PORT=3306
      - DB_USER=root
      - DB_PASSWORD=password
      - DB_NAME=lihatin_go
      - JWT_SECRET=your-production-jwt-secret
    depends_on:
      - db
    volumes:
      - ./.env:/root/.env

  db:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=password
      - MYSQL_DATABASE=lihatin_go
    ports:
      - "3306:3306"
    volumes:
      - db_data:/var/lib/mysql

volumes:
  db_data:
```

Deploy with Docker:

```bash
docker-compose up -d
```

### 7. Health Checks

The application provides several endpoints for health monitoring:

- `GET /api/health` - Basic health check
- `GET /api/auth/profile` - Authentication health (requires token)
- Admin endpoints for user and security monitoring

### 8. Troubleshooting

#### Common Issues

1. **Database Connection Issues**
   - Verify database credentials
   - Check firewall settings
   - Ensure database server is running

2. **Migration Errors**
   - Check database permissions
   - Verify migration files exist
   - Check for partial migrations

3. **Authentication Issues**
   - Verify JWT secret configuration
   - Check token expiration settings
   - Validate email service configuration

#### Logs and Debugging

- Application logs all requests via activity logger
- Check database logs for query issues
- Monitor failed login attempts for security issues

### 9. Scaling Considerations

#### Horizontal Scaling
- Use load balancer (nginx, HAProxy)
- Ensure stateless application design
- Configure session management appropriately

#### Database Scaling
- Consider read replicas for read-heavy workloads
- Implement database connection pooling
- Monitor query performance

### 10. Support and Maintenance

#### Regular Tasks
- Update dependencies regularly
- Monitor security advisories
- Review access logs for suspicious activity
- Backup configuration and data

#### Performance Monitoring
- Track API response times
- Monitor database query performance
- Watch memory and CPU usage
- Set up alerting for system issues

---

This deployment guide provides a comprehensive approach to getting Lihatin-Go running in production. Adjust the configuration based on your specific infrastructure requirements.
