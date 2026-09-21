package service_test

import (
	"context"
	"testing"

	"golang-dh/internal/modules/company/dto"
	"golang-dh/internal/modules/company/repository"
	"golang-dh/internal/modules/company/service"
)

func TestCompanyService_GetAllCompanies(t *testing.T) {
	mockData := []dto.CompanyItem{
		{ID: 1, Name: "qtera mandiri"},
		{ID: 2, Name: "bunda mulia"},
	}
	fakeRepo := repository.NewCompanyRepositoryFake(mockData)
	svc := service.NewCompanyService(fakeRepo)

	ctx := context.Background()
	result, err := svc.GetAllCompanies(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 companies, got %d", len(result))
	}
	if result[0].Name != "qtera mandiri" {
		t.Errorf("expected qtera mandiri, got %s", result[0].Name)
	}
}
