package repository

import (
	"context"
	"errors"
	"strings"

	"golang-dh/internal/modules/asset/dto"
)

// AssetRepositoryFake is an in-memory mock repository for asset unit tests.
type AssetRepositoryFake struct {
	Items []dto.AssetItem
}

func NewAssetRepositoryFake(initial []dto.AssetItem) *AssetRepositoryFake {
	return &AssetRepositoryFake{Items: initial}
}

func (f *AssetRepositoryFake) List(ctx context.Context, filter dto.AssetListFilter) ([]dto.AssetItem, int, error) {
	var result []dto.AssetItem
	for _, it := range f.Items {
		if filter.Perusahaan != "" && !strings.EqualFold(it.Perusahaan, filter.Perusahaan) {
			continue
		}
		if filter.Search != "" {
			match := strings.Contains(strings.ToLower(it.Name), strings.ToLower(filter.Search)) ||
				strings.Contains(strings.ToLower(it.AssetID), strings.ToLower(filter.Search))
			if !match {
				continue
			}
		}
		result = append(result, it)
	}
	return result, len(result), nil
}

func (f *AssetRepositoryFake) GetByID(ctx context.Context, assetID string) (*dto.AssetItem, error) {
	for _, it := range f.Items {
		if strings.EqualFold(it.AssetID, assetID) {
			return &it, nil
		}
	}
	return nil, errors.New("asset not found")
}

func (f *AssetRepositoryFake) GetOptions(ctx context.Context, perusahaan string) (dto.AssetOptionsResponse, error) {
	return dto.AssetOptionsResponse{
		Categories: []string{"IT Equipment"},
		Locations:  []string{"Warehouse A"},
		Brands:     []string{"Dell"},
	}, nil
}

func (f *AssetRepositoryFake) Create(ctx context.Context, a dto.AssetItem) error {
	f.Items = append(f.Items, a)
	return nil
}

func (f *AssetRepositoryFake) Update(ctx context.Context, assetID string, fields map[string]any) error {
	for i, it := range f.Items {
		if strings.EqualFold(it.AssetID, assetID) {
			if name, ok := fields["name"].(string); ok {
				f.Items[i].Name = name
			}
			return nil
		}
	}
	return errors.New("asset not found")
}

func (f *AssetRepositoryFake) Delete(ctx context.Context, assetID string) error {
	var remaining []dto.AssetItem
	found := false
	for _, it := range f.Items {
		if strings.EqualFold(it.AssetID, assetID) {
			found = true
		} else {
			remaining = append(remaining, it)
		}
	}
	if !found {
		return errors.New("asset not found")
	}
	f.Items = remaining
	return nil
}

func (f *AssetRepositoryFake) Exists(ctx context.Context, assetID string) (bool, error) {
	for _, it := range f.Items {
		if strings.EqualFold(it.AssetID, assetID) {
			return true, nil
		}
	}
	return false, nil
}
