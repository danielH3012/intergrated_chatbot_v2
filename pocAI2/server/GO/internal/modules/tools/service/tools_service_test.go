package service_test

import (
	"context"
	"testing"

	"golang-dh/internal/modules/tools/dto"
	"golang-dh/internal/modules/tools/repository"
	"golang-dh/internal/modules/tools/service"
)

func TestToolsService_RecordToolExecution(t *testing.T) {
	fakeRepo := repository.NewToolsRepositoryFake()
	svc := service.NewToolsService(fakeRepo)
	ctx := context.Background()

	// 1. Validation test
	_, err := svc.RecordToolExecution(ctx, dto.CreateToolsHistoryRequest{Tools: ""})
	if err == nil {
		t.Fatal("expected error on empty tools, got nil")
	}

	// 2. Success test
	id, err := svc.RecordToolExecution(ctx, dto.CreateToolsHistoryRequest{
		Tools:             "get_assets",
		Perusahaan:        "qtera mandiri",
		CreatedByUsername: "admin",
		CreatedByRole:     "admin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty record ID")
	}

	if len(fakeRepo.Records) != 1 {
		t.Fatalf("expected 1 record in fake repo, got %d", len(fakeRepo.Records))
	}
}
