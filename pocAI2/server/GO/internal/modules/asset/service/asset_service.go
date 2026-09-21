package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang-dh/internal/modules/asset/dto"
	"golang-dh/internal/modules/asset/repository"
)

// AssetService manages business operations on IT assets.
type AssetService struct {
	repo repository.AssetRepository
}

// NewAssetService creates a new AssetService.
func NewAssetService(repo repository.AssetRepository) *AssetService {
	return &AssetService{repo: repo}
}

// ListAssets returns paginated assets conforming to filters.
func (s *AssetService) ListAssets(ctx context.Context, filter dto.AssetListFilter) (dto.AssetListResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 10
	}

	assets, totalItems, err := s.repo.List(ctx, filter)
	if err != nil {
		return dto.AssetListResult{}, err
	}

	pageSize := filter.PageSize
	if filter.AllRecords {
		pageSize = totalItems
		if pageSize == 0 {
			pageSize = 1
		}
	}

	totalPages := (totalItems + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return dto.AssetListResult{
		Assets:     assets,
		Page:       filter.Page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}

// GetOptions retrieves distinct options for filtering in the UI.
func (s *AssetService) GetOptions(ctx context.Context, perusahaan string) (dto.AssetOptionsResponse, error) {
	return s.repo.GetOptions(ctx, perusahaan)
}

// GenerateAssetID creates a unique AST-XXXXXXXX formatted identifier.
func (s *AssetService) GenerateAssetID(ctx context.Context) (string, error) {
	for attempts := 0; attempts < 10; attempts++ {
		b := make([]byte, 4)
		_, err := rand.Read(b)
		if err != nil {
			return "", err
		}
		id := fmt.Sprintf("AST-%08X", b)
		exists, err := s.repo.Exists(ctx, id)
		if err != nil {
			return "", err
		}
		if !exists {
			return id, nil
		}
	}
	return "", errors.New("failed to generate unique asset ID after multiple attempts")
}

// CreateAsset validates and persists a new asset.
func (s *AssetService) CreateAsset(ctx context.Context, req dto.CreateAssetRequest, companyScope string) (string, error) {
	if strings.TrimSpace(req.Name) == "" {
		return "", errors.New("asset name is required")
	}
	if strings.TrimSpace(req.Category) == "" {
		return "", errors.New("category is required")
	}
	if strings.TrimSpace(req.Location) == "" {
		return "", errors.New("location is required")
	}

	assetID, err := s.GenerateAssetID(ctx)
	if err != nil {
		return "", err
	}

	perusahaan := strings.TrimSpace(companyScope)
	if perusahaan == "" {
		perusahaan = strings.TrimSpace(req.Perusahaan)
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}

	now := time.Now().Format(time.RFC3339)

	item := dto.AssetItem{
		AssetID:       assetID,
		Name:          strings.TrimSpace(req.Name),
		Category:      strings.TrimSpace(req.Category),
		Brand:         strings.TrimSpace(req.Brand),
		ModelType:     strings.TrimSpace(req.ModelType),
		PurchaseDate:  strings.TrimSpace(req.PurchaseDate),
		PurchasePrice: strings.TrimSpace(req.PurchasePrice),
		Location:      strings.TrimSpace(req.Location),
		CreatedAt:     now,
		Perusahaan:    perusahaan,
		Status:        status,
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return "", err
	}
	return assetID, nil
}

// UpdateAsset modifies fields on an existing asset with multi-tenant enforcement.
func (s *AssetService) UpdateAsset(ctx context.Context, assetID string, req dto.UpdateAssetRequest, callerRole, callerCompany string) error {
	existing, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return errors.New("asset not found")
	}

	if !canManageAsset(existing.Perusahaan, callerRole, callerCompany) {
		return fmt.Errorf("forbidden: asset belongs to company %q and cannot be modified by user in %q", existing.Perusahaan, callerCompany)
	}

	fields := make(map[string]any)
	if req.Name != nil {
		fields["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Category != nil {
		fields["category"] = strings.TrimSpace(*req.Category)
	}
	if req.Brand != nil {
		fields["brand"] = strings.TrimSpace(*req.Brand)
	}
	if req.ModelType != nil {
		fields["model_type"] = strings.TrimSpace(*req.ModelType)
	}
	if req.PurchaseDate != nil {
		fields["purchase_date"] = strings.TrimSpace(*req.PurchaseDate)
	}
	if req.PurchasePrice != nil {
		fields["purchase_price"] = strings.TrimSpace(*req.PurchasePrice)
	}
	if req.Location != nil {
		fields["location"] = strings.TrimSpace(*req.Location)
	}
	if req.Status != nil {
		fields["status"] = strings.TrimSpace(*req.Status)
	}

	return s.repo.Update(ctx, existing.AssetID, fields)
}

// DeleteAsset deletes an asset with multi-tenant enforcement.
func (s *AssetService) DeleteAsset(ctx context.Context, assetID string, callerRole, callerCompany string) error {
	existing, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return errors.New("asset not found")
	}

	if !canManageAsset(existing.Perusahaan, callerRole, callerCompany) {
		return fmt.Errorf("forbidden: asset belongs to company %q and cannot be deleted by user in %q", existing.Perusahaan, callerCompany)
	}

	return s.repo.Delete(ctx, existing.AssetID)
}

// Helper: check multi-tenancy access rights
func canManageAsset(assetCompany, userRole, userCompany string) bool {
	if strings.EqualFold(userRole, "superadmin") {
		return true
	}
	if strings.TrimSpace(userCompany) == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(assetCompany), strings.TrimSpace(userCompany))
}
