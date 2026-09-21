package repository

import (
	"context"

	"golang-dh/internal/modules/tools/dto"
)

// ToolsRepositoryFake is an in-memory repository for unit testing tools execution history.
type ToolsRepositoryFake struct {
	Records []dto.ToolsHistoryItem
}

func NewToolsRepositoryFake() *ToolsRepositoryFake {
	return &ToolsRepositoryFake{Records: []dto.ToolsHistoryItem{}}
}

func (f *ToolsRepositoryFake) Create(ctx context.Context, item dto.ToolsHistoryItem) error {
	f.Records = append(f.Records, item)
	return nil
}
