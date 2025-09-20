package repositories

import (
	"database/sql"

	"gorm.io/gorm"
)

// AuthRepository handles authentication-related database operations
type AuthRepository struct {
	db               *gorm.DB
	sqlDB            *sql.DB
	userAuthRepo     *UserAuthRepository
	authMethodRepo   *AuthMethodRepository
	apiKeyRepo       *APIKeyRepository
	loginAttemptRepo *LoginAttemptRepository

	// Public accessors for commonly used repositories
	APIKeyRepo       *APIKeyRepository
	LoginAttemptRepo *LoginAttemptRepository
}

// NewAuthRepository creates a new auth repository
func NewAuthRepository(gormDB *gorm.DB) *AuthRepository {
	// Get the underlying sql.DB from GORM
	sqlDB, err := gormDB.DB()
	if err != nil {
		panic("Failed to get SQL DB from GORM: " + err.Error())
	}

	apiKeyRepo := NewAPIKeyRepository(gormDB)
	loginAttemptRepo := NewLoginAttemptRepository(gormDB)

	return &AuthRepository{
		db:               gormDB,
		sqlDB:            sqlDB,
		userAuthRepo:     NewUserAuthRepository(gormDB),
		authMethodRepo:   NewAuthMethodRepository(gormDB),
		apiKeyRepo:       apiKeyRepo,
		loginAttemptRepo: loginAttemptRepo,

		// Public accessors
		APIKeyRepo:       apiKeyRepo,
		LoginAttemptRepo: loginAttemptRepo,
	}
}

// GetUserRepository returns a GORM-based user repository
func (r *AuthRepository) GetUserRepository() UserRepository {
	return NewUserRepository(r.db)
}

// GetUserAuthRepository returns the user auth repository
func (r *AuthRepository) GetUserAuthRepository() *UserAuthRepository {
	return r.userAuthRepo
}

// GetAuthMethodRepository returns the auth method repository
func (r *AuthRepository) GetAuthMethodRepository() *AuthMethodRepository {
	return r.authMethodRepo
}

// GetAPIKeyRepository returns the API key repository
func (r *AuthRepository) GetAPIKeyRepository() *APIKeyRepository {
	return r.apiKeyRepo
}

// GetLoginAttemptRepository returns the login attempt repository
func (r *AuthRepository) GetLoginAttemptRepository() *LoginAttemptRepository {
	return r.loginAttemptRepo
}
