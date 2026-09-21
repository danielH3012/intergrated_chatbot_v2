package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"golang-dh/internal/modules/tools/dto"
	"golang-dh/pkg/database"
)

// ToolsRepository defines data operations for public.tools_history.
type ToolsRepository interface {
	Create(ctx context.Context, item dto.ToolsHistoryItem) error
}

type toolsRepository struct {
	db database.PostgreSQLServicer
}

// NewToolsRepository creates a new ToolsRepository instance.
func NewToolsRepository(db database.PostgreSQLServicer) ToolsRepository {
	return &toolsRepository{db: db}
}

func (r *toolsRepository) Create(ctx context.Context, item dto.ToolsHistoryItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO public.tools_history (id, perusahaan, tools, created_by_user_id, created_by_username, created_by_role, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query, item.ID, item.Perusahaan, item.Tools, item.CreatedByUserID, item.CreatedByUsername, item.CreatedByRole, item.CreatedAt)
	return err
}
