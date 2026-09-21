package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"golang-dh/internal/modules/tools/dto"
	"golang-dh/internal/modules/tools/repository"
)

// ToolsService orchestrates tools history persistence.
type ToolsService struct {
	repo repository.ToolsRepository
}

// NewToolsService creates a new ToolsService instance.
func NewToolsService(repo repository.ToolsRepository) *ToolsService {
	return &ToolsService{repo: repo}
}

func (s *ToolsService) RecordToolExecution(ctx context.Context, req dto.CreateToolsHistoryRequest) (string, error) {
	tools := strings.TrimSpace(req.Tools)
	if tools == "" {
		return "", errors.New("tools name is required")
	}

	recordID := uuid.New().String()
	item := dto.ToolsHistoryItem{
		ID:                recordID,
		Perusahaan:        strings.TrimSpace(req.Perusahaan),
		Tools:             tools,
		CreatedByUserID:   strings.TrimSpace(req.CreatedByUserID),
		CreatedByUsername: strings.TrimSpace(req.CreatedByUsername),
		CreatedByRole:     strings.TrimSpace(req.CreatedByRole),
		CreatedAt:         time.Now(),
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return "", err
	}
	return recordID, nil
}
