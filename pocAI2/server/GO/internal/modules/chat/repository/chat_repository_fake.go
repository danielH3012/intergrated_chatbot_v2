package repository

import (
	"context"
	"strings"

	"golang-dh/internal/modules/chat/dto"
)

// ChatRepositoryFake provides an in-memory repository for unit testing conversation flow.
type ChatRepositoryFake struct {
	Messages []dto.ChatMessageItem
}

func NewChatRepositoryFake() *ChatRepositoryFake {
	return &ChatRepositoryFake{Messages: []dto.ChatMessageItem{}}
}

func (f *ChatRepositoryFake) SaveMessage(ctx context.Context, msg dto.ChatMessageItem) error {
	f.Messages = append(f.Messages, msg)
	return nil
}

func (f *ChatRepositoryFake) GetHistory(ctx context.Context, username string) ([]dto.ChatMessageItem, error) {
	var result []dto.ChatMessageItem
	for _, m := range f.Messages {
		if strings.EqualFold(m.Username, username) {
			result = append(result, m)
		}
	}
	return result, nil
}
