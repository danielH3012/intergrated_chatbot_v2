package service

import (
	"context"

	"golang-dh/internal/modules/company/dto"
	"golang-dh/internal/modules/company/repository"
)

// CompanyService handles business logic for company/tenant operations.
type CompanyService struct {
	repo repository.CompanyRepository
}

// NewCompanyService initializes a new CompanyService.
func NewCompanyService(repo repository.CompanyRepository) *CompanyService {
	return &CompanyService{repo: repo}
}

// GetAllCompanies retrieves all registered companies.
func (s *CompanyService) GetAllCompanies(ctx context.Context) ([]dto.CompanyItem, error) {
	return s.repo.List(ctx)
}

// FindByID retrieves a company by its ID.
func (s *CompanyService) FindByID(ctx context.Context, id int64) (*dto.CompanyItem, error) {
	return s.repo.GetByID(ctx, id)
}

// FindByName retrieves a company by its name.
func (s *CompanyService) FindByName(ctx context.Context, name string) (*dto.CompanyItem, error) {
	return s.repo.GetByName(ctx, name)
}
