package dto

import "time"

// RegisterRequest represents payload to create a new user account.
type RegisterRequest struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	Role         string `json:"role"`
	Company      string `json:"company"`
	IdPerusahaan int    `json:"id_perusahaan"`
}

// LoginRequest represents login credentials.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserResponse represents public user information without password hash.
type UserResponse struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	Company      string    `json:"company"`
	IdPerusahaan int       `json:"id_perusahaan"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuthResponse returns token and user payload.
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}
