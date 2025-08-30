# Environment Configuration Usage

This project uses centralized environment variable configuration through the `utils` package. The configuration system safely handles `.env` file loading with proper error handling.

## Available Functions

### `GetEnvOrDefault(key, defaultValue string) string`
Returns environment variable value or default if not set.

```go
// Example usage
port := utils.GetEnvOrDefault("PORT", "8080")
apiKey := utils.GetEnvOrDefault("API_KEY", "")
```

### `GetRequiredEnv(key string) string`
Returns environment variable value or panics if not set. Use for critical configuration.

```go
// Example usage
databaseURL := utils.GetRequiredEnv(utils.EnvDatabaseURL)
```

### `GetEnvAsInt(key string, defaultValue int) int`
Returns environment variable as integer or default.

```go
// Example usage
maxConnections := utils.GetEnvAsInt("MAX_CONNECTIONS", 100)
```

### `GetEnvAsBool(key string, defaultValue bool) bool`
Returns environment variable as boolean or default.

```go
// Example usage
debugMode := utils.GetEnvAsBool("DEBUG", false)
enableSSL := utils.GetEnvAsBool("ENABLE_SSL", true)
```

## Environment Variable Constants

The package provides constants for commonly used environment variables:

- `utils.EnvDatabaseURL` - "DATABASE_URL"
- `utils.EnvSMTPHost` - "SMTP_HOST"
- `utils.EnvSMTPPort` - "SMTP_PORT"
- `utils.EnvSMTPUser` - "SMTP_USER"
- `utils.EnvSMTPPass` - "SMTP_PASS"
- `utils.EnvFromEmail` - "FROM_EMAIL"
- `utils.EnvFromName` - "FROM_NAME"
- `utils.EnvJWTSecret` - "JWT_SECRET"

## Error Handling

- The system loads `.env` file automatically on first use
- If `.env` file is missing, it logs a warning but continues (environment variables can be set elsewhere)
- For required variables, use `GetRequiredEnv()` which will terminate the application if the variable is not set
- For optional variables, use `GetEnvOrDefault()` with appropriate defaults

## Migration from Direct os.Getenv Usage

### Before:
```go
err := godotenv.Load()
if err != nil {
    log.Fatal("Error loading .env file")
}
dsn := os.Getenv("DATABASE_URL")
if dsn == "" {
    log.Fatal("DATABASE_URL config is required")
}
```

### After:
```go
dsn := utils.GetRequiredEnv(utils.EnvDatabaseURL)
```
