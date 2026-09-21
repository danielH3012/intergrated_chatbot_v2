package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"golang-dh/internal/modules/chat/dto"
	"golang-dh/internal/modules/chat/repository"
)

// ChatService coordinates chat persistence and conversation retrieval.
type ChatService struct {
	repo repository.ChatRepository
}

// NewChatService creates a new ChatService.
func NewChatService(repo repository.ChatRepository) *ChatService {
	return &ChatService{repo: repo}
}

// PersistMessage stores an individual chat message in the repository.
func (s *ChatService) PersistMessage(ctx context.Context, msg dto.ChatMessageItem) error {
	if strings.TrimSpace(msg.ID) == "" {
		msg.ID = uuid.New().String()
	}
	if strings.TrimSpace(msg.CreatedAt) == "" {
		msg.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	}
	return s.repo.SaveMessage(ctx, msg)
}

// GetUserHistory retrieves and formats chat messages for a specific user.
func (s *ChatService) GetUserHistory(ctx context.Context, username string) ([]dto.ChatHistoryResponse, error) {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return nil, errors.New("username is required")
	}

	messages, err := s.repo.GetHistory(ctx, trimmed)
	if err != nil {
		return nil, err
	}

	var response []dto.ChatHistoryResponse
	for _, m := range messages {
		displayName := m.Username
		if !m.IsUser && m.Role == "system" {
			displayName = "QTERA AI"
		}

		response = append(response, dto.ChatHistoryResponse{
			ID:     m.ID,
			Chat:   m.Chat,
			User:   m.IsUser,
			Date:   m.CreatedAt,
			Attach: m.Attachment,
			UserObj: map[string]any{
				"name": displayName,
				"role": m.Role,
			},
		})
	}
	if response == nil {
		response = []dto.ChatHistoryResponse{}
	}
	return response, nil
}
