package dto

// Auth-related Data Transfer Objects (DTOs)

// LoginRequest represents the login request payload
type LoginRequest struct {
	EmailOrUsername string `json:"email_or_username" label:"Email atau Username" binding:"required,min=3,max=100"`
	Password        string `json:"password" label:"Kata Sandi" binding:"required,min=8,max=50,pwdcomplex"`
}

// LoginResponse represents the successful login response
type LoginResponse struct {
	Token TokenResponse    `json:"token"`
	User  UserProfile      `json:"user"`
	Auth  UserAuthResponse `json:"auth"`
}

// TokenResponse represents access and refresh tokens
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// UserProfile represents user information returned after login (without sensitive data)
type UserProfile struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	IsPremium bool   `json:"is_premium"`
	CreatedAt string `json:"created_at"`
}

// RegisterRequest represents the user registration request payload
type RegisterRequest struct {
	FirstName string `json:"first_name" binding:"required,min=2,max=50"`
	LastName  string `json:"last_name" binding:"required,min=2,max=50"`
	Username  string `json:"username" binding:"required,min=3,max=30,alphanum"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8,max=50,pwdcomplex"`
}

// UpdateProfileRequest represents the user profile update request payload
type UpdateProfileRequest struct {
	FirstName *string `json:"first_name" binding:"omitempty,min=2,max=50"`
	LastName  *string `json:"last_name" binding:"omitempty,min=2,max=50"`
	Username  *string `json:"username" binding:"omitempty,min=3,max=30,alphanum"`
	Email     *string `json:"email" binding:"omitempty,email"`
	Avatar    *string `json:"avatar" binding:"omitempty,url"`
}

// DeleteAccountRequest represents the request to delete a user account
type DeleteAccountRequest struct {
	ID string `json:"id" binding:"required,uuid7" label:"ID Pengguna" uri:"id"`
}

// UserAuthResponse represents user authentication details (without sensitive data)
type UserAuthResponse struct {
	ID              string `json:"id"`
	UserID          string `json:"user_id"`
	IsEmailVerified bool   `json:"is_email_verified"`
	IsTOTPEnabled   bool   `json:"is_totp_enabled"`
	LastLoginAt     string `json:"last_login_at"`
}
