package models

import "time"

// User maps to public.users in PostgreSQL.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // "admin" | "operator" | "viewer"
	Company      string    `json:"company,omitempty"`
	IdPerusahaan int       `json:"id_perusahaan,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
