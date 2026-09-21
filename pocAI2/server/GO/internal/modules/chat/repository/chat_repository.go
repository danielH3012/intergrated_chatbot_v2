package repository

import (
	"context"
	"fmt"

	"golang-dh/internal/modules/chat/dto"
	"golang-dh/pkg/database"
)

// ChatRepository defines database access for public.chat.
type ChatRepository interface {
	SaveMessage(ctx context.Context, msg dto.ChatMessageItem) error
	GetHistory(ctx context.Context, username string) ([]dto.ChatMessageItem, error)
}

type chatRepository struct {
	db database.PostgreSQLServicer
}

// NewChatRepository creates a new ChatRepository.
func NewChatRepository(db database.PostgreSQLServicer) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) SaveMessage(ctx context.Context, msg dto.ChatMessageItem) error {
	query := `
		INSERT INTO public.chat (id, "user", chat, created_at, user_id, username, role, models,
		                         attachment_name, attachment_url, attachment_type, attachment_size, attachment_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Exec(ctx, query,
		msg.ID, msg.IsUser, msg.Chat, msg.CreatedAt, msg.UserID, msg.Username, msg.Role, msg.Models,
		msg.AttachmentName, msg.AttachmentURL, msg.AttachmentType, msg.AttachmentSize, msg.AttachmentText,
	)
	return err
}

func (r *chatRepository) GetHistory(ctx context.Context, username string) ([]dto.ChatMessageItem, error) {
	query := `
		SELECT id, "user", chat, created_at, user_id, username, role, models,
		       attachment_name, attachment_url, attachment_type, attachment_size, attachment_text
		FROM public.chat
		WHERE username ILIKE $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(ctx, query, username)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat history: %w", err)
	}
	defer rows.Close()

	var history []dto.ChatMessageItem
	for rows.Next() {
		var item dto.ChatMessageItem
		if err := rows.Scan(
			&item.ID, &item.IsUser, &item.Chat, &item.CreatedAt, &item.UserID, &item.Username, &item.Role, &item.Models,
			&item.AttachmentName, &item.AttachmentURL, &item.AttachmentType, &item.AttachmentSize, &item.AttachmentText,
		); err != nil {
			continue
		}

		if item.AttachmentName != nil && *item.AttachmentName != "" {
			url := ""
			if item.AttachmentURL != nil {
				url = *item.AttachmentURL
			}
			tType := ""
			if item.AttachmentType != nil {
				tType = *item.AttachmentType
			}
			var size int64 = 0
			if item.AttachmentSize != nil {
				size = *item.AttachmentSize
			}
			item.Attachment = &dto.AttachmentInfo{
				Name: *item.AttachmentName,
				URL:  url,
				Type: tType,
				Size: size,
			}
		}

		history = append(history, item)
	}
	if history == nil {
		history = []dto.ChatMessageItem{}
	}
	return history, nil
}
