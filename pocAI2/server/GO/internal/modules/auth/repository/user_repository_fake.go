package repository

import (
	"context"
	"errors"
	"strings"

	"golang-dh/models"
)

// UserRepositoryFake provides an in-memory repository for user unit testing.
type UserRepositoryFake struct {
	Users []models.User
}

func NewUserRepositoryFake(initial []models.User) *UserRepositoryFake {
	return &UserRepositoryFake{Users: initial}
}

func (f *UserRepositoryFake) Exists(ctx context.Context, username, email string) (bool, error) {
	for _, u := range f.Users {
		if strings.EqualFold(u.Username, username) || strings.EqualFold(u.Email, email) {
			return true, nil
		}
	}
	return false, nil
}

func (f *UserRepositoryFake) ResolvePerusahaanName(ctx context.Context, idPerusahaan int) (string, error) {
	if idPerusahaan == 1 {
		return "qtera mandiri", nil
	}
	return "Test Company", nil
}

func (f *UserRepositoryFake) ResolvePerusahaanID(ctx context.Context, companyName string) (int, error) {
	return 1, nil
}

func (f *UserRepositoryFake) Create(ctx context.Context, user *models.User) error {
	f.Users = append(f.Users, *user)
	return nil
}

func (f *UserRepositoryFake) FindByUsernameOrEmail(ctx context.Context, identifier string) (*models.User, error) {
	for _, u := range f.Users {
		if strings.EqualFold(u.Username, identifier) || strings.EqualFold(u.Email, identifier) {
			return &u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (f *UserRepositoryFake) FindByID(ctx context.Context, id string) (*models.User, error) {
	for _, u := range f.Users {
		if u.ID == id {
			return &u, nil
		}
	}
	return nil, errors.New("user not found")
}
