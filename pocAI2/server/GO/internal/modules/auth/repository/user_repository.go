package repository

import (
	"context"
	"time"

	"golang-dh/models"
	"golang-dh/pkg/database"
)

// UserRepository defines operations on the public.users database table.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByUsernameOrEmail(ctx context.Context, identifier string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	Exists(ctx context.Context, username, email string) (bool, error)
	ResolvePerusahaanName(ctx context.Context, idPerusahaan int) (string, error)
	ResolvePerusahaanID(ctx context.Context, companyName string) (int, error)
}

type userRepository struct {
	db database.PostgreSQLServicer
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db database.PostgreSQLServicer) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Exists(ctx context.Context, username, email string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM public.users WHERE username = $1 OR email = $2`, username, email).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *userRepository) ResolvePerusahaanName(ctx context.Context, idPerusahaan int) (string, error) {
	var name string
	err := r.db.QueryRow(ctx, `SELECT nama_perusahaan FROM public.perusahaan WHERE id_perusahaan = $1 LIMIT 1`, idPerusahaan).Scan(&name)
	return name, err
}

func (r *userRepository) ResolvePerusahaanID(ctx context.Context, companyName string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, `SELECT id_perusahaan FROM public.perusahaan WHERE nama_perusahaan = $1 LIMIT 1`, companyName).Scan(&id)
	return id, err
}

func (r *userRepository) Create(ctx context.Context, u *models.User) error {
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = time.Now()
	}

	query := `
		INSERT INTO public.users (id, username, email, password_hash, role, company, id_perusahaan, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query, u.ID, u.Username, u.Email, u.PasswordHash, u.Role, u.Company, u.IdPerusahaan, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		// Fallback without id_perusahaan column if table schema differs
		fallbackQuery := `
			INSERT INTO public.users (id, username, email, password_hash, role, company, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = r.db.Exec(ctx, fallbackQuery, u.ID, u.Username, u.Email, u.PasswordHash, u.Role, u.Company, u.CreatedAt, u.UpdatedAt)
	}
	return err
}

func (r *userRepository) FindByUsernameOrEmail(ctx context.Context, identifier string) (*models.User, error) {
	var u models.User
	query := `
		SELECT id, username, email, password_hash, role, company, COALESCE(id_perusahaan, 0), created_at, updated_at
		FROM public.users
		WHERE username = $1 OR email = $2
		LIMIT 1
	`
	err := r.db.QueryRow(ctx, query, identifier, identifier).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.Company, &u.IdPerusahaan, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		// Fallback without id_perusahaan
		fallbackQuery := `
			SELECT id, username, email, password_hash, role, company, created_at, updated_at
			FROM public.users
			WHERE username = $1 OR email = $2
			LIMIT 1
		`
		err = r.db.QueryRow(ctx, fallbackQuery, identifier, identifier).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.Company, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
	}
	return &u, nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	query := `
		SELECT id, username, email, role, company, COALESCE(id_perusahaan, 0), created_at
		FROM public.users
		WHERE id = $1
		LIMIT 1
	`
	err := r.db.QueryRow(ctx, query, id).Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Company, &u.IdPerusahaan, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
