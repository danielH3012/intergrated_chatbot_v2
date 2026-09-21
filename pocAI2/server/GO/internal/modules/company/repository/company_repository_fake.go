package repository

import (
	"context"
	"errors"
	"strings"

	"golang-dh/internal/modules/company/dto"
)

// CompanyRepositoryFake is an in-memory mock implementation for unit testing.
type CompanyRepositoryFake struct {
	Items []dto.CompanyItem
}

func NewCompanyRepositoryFake(initial []dto.CompanyItem) *CompanyRepositoryFake {
	return &CompanyRepositoryFake{Items: initial}
}

func (f *CompanyRepositoryFake) List(ctx context.Context) ([]dto.CompanyItem, error) {
	return f.Items, nil
}

func (f *CompanyRepositoryFake) GetByID(ctx context.Context, id int64) (*dto.CompanyItem, error) {
	for _, it := range f.Items {
		if it.ID == id {
			return &it, nil
		}
	}
	return nil, errors.New("company not found")
}

func (f *CompanyRepositoryFake) GetByName(ctx context.Context, name string) (*dto.CompanyItem, error) {
	for _, it := range f.Items {
		if strings.EqualFold(it.Name, name) {
			return &it, nil
		}
	}
	return nil, errors.New("company not found")
}
