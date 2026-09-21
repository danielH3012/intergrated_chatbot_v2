package service_test

import (
	"context"
	"testing"

	"golang-dh/internal/modules/asset/dto"
	"golang-dh/internal/modules/asset/repository"
	"golang-dh/internal/modules/asset/service"
)

func TestAssetService_Lifecycle(t *testing.T) {
	fakeRepo := repository.NewAssetRepositoryFake(nil)
	svc := service.NewAssetService(fakeRepo)
	ctx := context.Background()

	// 1. Validation test on creation
	_, err := svc.CreateAsset(ctx, dto.CreateAssetRequest{Name: ""}, "qtera mandiri")
	if err == nil {
		t.Fatal("expected error on empty asset name, got nil")
	}

	// 2. Success creation test
	assetID, err := svc.CreateAsset(ctx, dto.CreateAssetRequest{
		Name:     "MacBook Pro 16",
		Category: "IT Equipment > Laptop",
		Brand:    "Apple",
		Location: "Office 2nd Floor",
	}, "qtera mandiri")
	if err != nil {
		t.Fatalf("unexpected error creating asset: %v", err)
	}
	if len(assetID) == 0 {
		t.Fatal("expected non-empty asset ID")
	}

	// 3. List test
	res, err := svc.ListAssets(ctx, dto.AssetListFilter{
		Perusahaan: "qtera mandiri",
	})
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if res.TotalItems != 1 {
		t.Fatalf("expected 1 asset, got %d", res.TotalItems)
	}

	// 4. Multi-tenancy isolation test: unauthorized delete
	err = svc.DeleteAsset(ctx, assetID, "operator", "other company")
	if err == nil {
		t.Fatal("expected forbidden error on cross-tenant delete, got nil")
	}

	// 5. Authorized delete
	err = svc.DeleteAsset(ctx, assetID, "operator", "qtera mandiri")
	if err != nil {
		t.Fatalf("unexpected error on authorized delete: %v", err)
	}

	// 6. Verify empty list
	resAfter, _ := svc.ListAssets(ctx, dto.AssetListFilter{Perusahaan: "qtera mandiri"})
	if resAfter.TotalItems != 0 {
		t.Fatalf("expected 0 assets after delete, got %d", resAfter.TotalItems)
	}
}
