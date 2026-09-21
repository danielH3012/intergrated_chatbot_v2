package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"golang-dh/internal/modules/auth/dto"
	"golang-dh/internal/modules/auth/repository"
	"golang-dh/middleware"
	"golang-dh/models"
)

// AuthService coordinates credentials, tokens, and account registration.
type AuthService struct {
	repo repository.UserRepository
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(repo repository.UserRepository) *AuthService {
	return &AuthService{repo: repo}
}

// Register validates and creates a new user account.
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)
	password := req.Password

	if username == "" {
		return nil, errors.New("username is required")
	}
	if email == "" {
		return nil, errors.New("email is required")
	}
	if len(password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}

	// Resolve company and id_perusahaan
	company := strings.TrimSpace(req.Company)
	idPerusahaan := req.IdPerusahaan

	if idPerusahaan > 0 && company == "" {
		if name, err := s.repo.ResolvePerusahaanName(ctx, idPerusahaan); err == nil && name != "" {
			company = name
		}
	} else if company != "" && idPerusahaan == 0 {
		if id, err := s.repo.ResolvePerusahaanID(ctx, company); err == nil && id > 0 {
			idPerusahaan = id
		}
	}

	exists, err := s.repo.Exists(ctx, username, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("username or email already in use")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to encrypt password")
	}

	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "operator"
	}

	now := time.Now()
	newUser := models.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         role,
		Company:      company,
		IdPerusahaan: idPerusahaan,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, &newUser); err != nil {
		return nil, errors.New("failed to persist user account")
	}

	token, err := middleware.GenerateJWT(&newUser)
	if err != nil {
		return nil, errors.New("failed to generate authentication token")
	}

	return &dto.AuthResponse{
		Token: token,
		User: dto.UserResponse{
			ID:           newUser.ID,
			Username:     newUser.Username,
			Email:        newUser.Email,
			Role:         newUser.Role,
			Company:      newUser.Company,
			IdPerusahaan: newUser.IdPerusahaan,
			CreatedAt:    newUser.CreatedAt,
		},
	}, nil
}

// Login verifies credentials and returns a JWT token.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	identifier := strings.TrimSpace(req.Username)
	password := req.Password

	if identifier == "" || password == "" {
		return nil, errors.New("username/email and password are required")
	}

	user, err := s.repo.FindByUsernameOrEmail(ctx, identifier)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	token, err := middleware.GenerateJWT(user)
	if err != nil {
		return nil, errors.New("failed to generate authentication token")
	}

	return &dto.AuthResponse{
		Token: token,
		User: dto.UserResponse{
			ID:           user.ID,
			Username:     user.Username,
			Email:        user.Email,
			Role:         user.Role,
			Company:      user.Company,
			IdPerusahaan: user.IdPerusahaan,
			CreatedAt:    user.CreatedAt,
		},
	}, nil
}

// GetMe retrieves the current authenticated user's profile.
func (s *AuthService) GetMe(ctx context.Context, userID string) (*dto.UserResponse, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return &dto.UserResponse{
		ID:           user.ID,
		Username:     user.Username,
		Email:        user.Email,
		Role:         user.Role,
		Company:      user.Company,
		IdPerusahaan: user.IdPerusahaan,
		CreatedAt:    user.CreatedAt,
	}, nil
}
