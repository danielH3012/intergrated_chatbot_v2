package service_test

import (
	"context"
	"testing"

	"golang-dh/internal/modules/chat/dto"
	"golang-dh/internal/modules/chat/repository"
	"golang-dh/internal/modules/chat/service"
)

func TestChatService_PersistAndRetrieve(t *testing.T) {
	fakeRepo := repository.NewChatRepositoryFake()
	svc := service.NewChatService(fakeRepo)
	ctx := context.Background()

	// 1. Validation test
	_, err := svc.GetUserHistory(ctx, "")
	if err == nil {
		t.Fatal("expected error on empty username, got nil")
	}

	// 2. Persist message
	err = svc.PersistMessage(ctx, dto.ChatMessageItem{
		Username: "john_doe",
		Chat:     "Hello, what assets do I have?",
		IsUser:   true,
		Role:     "operator",
	})
	if err != nil {
		t.Fatalf("unexpected error persisting message: %v", err)
	}

	// 3. Persist AI response
	err = svc.PersistMessage(ctx, dto.ChatMessageItem{
		Username: "john_doe",
		Chat:     "You have 3 active assets.",
		IsUser:   false,
		Role:     "system",
	})
	if err != nil {
		t.Fatalf("unexpected error persisting AI response: %v", err)
	}

	// 4. Retrieve history
	history, err := svc.GetUserHistory(ctx, "john_doe")
	if err != nil {
		t.Fatalf("unexpected error retrieving history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}

	if history[1].UserObj["name"] != "QTERA AI" {
		t.Errorf("expected system display name QTERA AI, got %v", history[1].UserObj["name"])
	}
}
