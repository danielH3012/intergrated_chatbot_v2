package controllers

import (
	"context"
	"strings"
	"time"

	"golang-dh/middleware"
	"golang-dh/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	Role         string `json:"role"`
	Company      string `json:"company"`
	IdPerusahaan int    `json:"id_perusahaan"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	Company      string    `json:"company,omitempty"`
	IdPerusahaan int       `json:"id_perusahaan,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

func Register(c *fiber.Ctx) error {
	var body RegisterRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	body.Username = strings.TrimSpace(body.Username)
	body.Email = strings.ToLower(strings.TrimSpace(body.Email))
	body.Password = strings.TrimSpace(body.Password)
	body.Company = strings.TrimSpace(body.Company)
	body.Role = strings.ToLower(strings.TrimSpace(body.Role))

	if body.Username == "" || body.Email == "" || body.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "username, email, and password are required",
		})
	}
	if len(body.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "password must be at least 6 characters long",
		})
	}
	if body.Role == "" {
		body.Role = "operator"
	}
	if body.Role != "admin" && body.Role != "operator" && body.Role != "viewer" {
		body.Role = "operator"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// If IdPerusahaan is given but Company is empty, lookup company name
	if body.IdPerusahaan != 0 && body.Company == "" {
		var compName string
		row := DB.QueryRow(ctx, `SELECT nama_perusahaan FROM perusahaan WHERE id_perusahaan = $1 LIMIT 1`, body.IdPerusahaan)
		if err := row.Scan(&compName); err == nil && compName != "" {
			body.Company = compName
		}
	} else if body.Company != "" && body.IdPerusahaan == 0 {
		var idComp int
		row := DB.QueryRow(ctx, `SELECT id_perusahaan FROM perusahaan WHERE nama_perusahaan = $1 LIMIT 1`, body.Company)
		if err := row.Scan(&idComp); err == nil && idComp != 0 {
			body.IdPerusahaan = idComp
		}
	}

	// Check duplicate username / email
	var existingUsername, existingEmail string
	row := DB.QueryRow(ctx,
		`SELECT username, email FROM users WHERE username=$1 OR email=$2 LIMIT 1`,
		body.Username, body.Email,
	)
	if err := row.Scan(&existingUsername, &existingEmail); err == nil {
		if strings.EqualFold(existingUsername, body.Username) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "username is already taken",
			})
		}
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "email is already registered",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to hash password",
		})
	}

	now := time.Now()
	newUser := models.User{
		ID:           uuid.New().String(),
		Username:     body.Username,
		Email:        body.Email,
		PasswordHash: string(hashedPassword),
		Role:         body.Role,
		Company:      body.Company,
		IdPerusahaan: body.IdPerusahaan,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err = DB.Exec(ctx,
		`INSERT INTO users (id, username, email, password_hash, role, company, id_perusahaan, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		newUser.ID, newUser.Username, newUser.Email, newUser.PasswordHash,
		newUser.Role, newUser.Company, newUser.IdPerusahaan, newUser.CreatedAt, newUser.UpdatedAt,
	)
	if err != nil {
		// Fallback in case id_perusahaan column doesn't exist
		_, err = DB.Exec(ctx,
			`INSERT INTO users (id, username, email, password_hash, role, company, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			newUser.ID, newUser.Username, newUser.Email, newUser.PasswordHash,
			newUser.Role, newUser.Company, newUser.CreatedAt, newUser.UpdatedAt,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to create user account",
			})
		}
	}

	token, err := middleware.GenerateJWT(&newUser)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate authentication token",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"token": token,
		"user": UserResponse{
			ID:           newUser.ID,
			Username:     newUser.Username,
			Email:        newUser.Email,
			Role:         newUser.Role,
			Company:      newUser.Company,
			IdPerusahaan: newUser.IdPerusahaan,
			CreatedAt:    newUser.CreatedAt,
		},
	})
}

func Login(c *fiber.Ctx) error {
	var body LoginRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	body.Username = strings.TrimSpace(body.Username)
	body.Password = strings.TrimSpace(body.Password)

	if body.Username == "" || body.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "username/email and password are required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	row := DB.QueryRow(ctx,
		`SELECT id, username, email, password_hash, role, company, COALESCE(id_perusahaan, 0), created_at, updated_at
		 FROM users WHERE username=$1 OR email=$2 LIMIT 1`,
		body.Username, strings.ToLower(body.Username),
	)
	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.Company, &user.IdPerusahaan, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		// Fallback without id_perusahaan column
		rowLegacy := DB.QueryRow(ctx,
			`SELECT id, username, email, password_hash, role, company, created_at, updated_at
			 FROM users WHERE username=$1 OR email=$2 LIMIT 1`,
			body.Username, strings.ToLower(body.Username),
		)
		if errLegacy := rowLegacy.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role, &user.Company, &user.CreatedAt, &user.UpdatedAt); errLegacy != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid username/email or password",
			})
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid username/email or password",
		})
	}

	token, err := middleware.GenerateJWT(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate authentication token",
		})
	}

	return c.JSON(fiber.Map{
		"token": token,
		"user": UserResponse{
			ID:           user.ID,
			Username:     user.Username,
			Email:        user.Email,
			Role:         user.Role,
			Company:      user.Company,
			IdPerusahaan: user.IdPerusahaan,
			CreatedAt:    user.CreatedAt,
		},
	})
}

func GetMe(c *fiber.Ctx) error {
	userVal := c.Locals("user")
	if userVal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	claims, ok := userVal.(*middleware.JWTClaims)
	if !ok || claims == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid session claims",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	row := DB.QueryRow(ctx,
		`SELECT id, username, email, role, company, COALESCE(id_perusahaan, 0), created_at FROM users WHERE id=$1`,
		claims.UserID,
	)
	if err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.Company, &user.IdPerusahaan, &user.CreatedAt); err == nil {
		return c.JSON(fiber.Map{
			"user": UserResponse{
				ID:           user.ID,
				Username:     user.Username,
				Email:        user.Email,
				Role:         user.Role,
				Company:      user.Company,
				IdPerusahaan: user.IdPerusahaan,
				CreatedAt:    user.CreatedAt,
			},
		})
	}

	// Fallback to JWT claims
	return c.JSON(fiber.Map{
		"user": UserResponse{
			ID:           claims.UserID,
			Username:     claims.Username,
			Email:        claims.Email,
			Role:         claims.Role,
			Company:      claims.Company,
			IdPerusahaan: claims.IdPerusahaan,
		},
	})
}
