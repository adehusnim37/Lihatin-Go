package models

// LoginRequest represents the login request payload
type LoginRequest struct {
	EmailOrUsername string `json:"email_or_username" validate:"required,min=3,max=100"`
	Password        string `json:"password" validate:"required,min=8,max=50"`
}

// LoginResponse represents the successful login response
type LoginResponse struct {
	User    UserProfile `json:"user"`
	Token   string      `json:"token,omitempty"` // JWT token (if implementing JWT)
	Message string      `json:"message"`
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
