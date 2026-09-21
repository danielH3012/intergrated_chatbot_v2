package handler

import (
	"golang-dh/internal/modules/auth/service"
)

// AuthHandler manages HTTP authentication requests.
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}
