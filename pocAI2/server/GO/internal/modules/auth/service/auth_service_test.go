package service_test

import (
	"context"
	"os"
	"testing"

	"golang-dh/internal/modules/auth/dto"
	"golang-dh/internal/modules/auth/repository"
	"golang-dh/internal/modules/auth/service"
)

func TestAuthService_RegisterAndLogin(t *testing.T) {
	_ = os.Setenv("JWT_SECRET", "test-secret-key-12345678901234567890")
	fakeRepo := repository.NewUserRepositoryFake(nil)
	svc := service.NewAuthService(fakeRepo)
	ctx := context.Background()

	// 1. Validation test on registration
	_, err := svc.Register(ctx, dto.RegisterRequest{
		Username: "",
		Email:    "test@test.com",
		Password: "123",
	})
	if err == nil {
		t.Fatal("expected error on invalid registration input, got nil")
	}

	// 2. Successful registration
	regRes, err := svc.Register(ctx, dto.RegisterRequest{
		Username: "newuser",
		Email:    "newuser@gmail.com",
		Password: "password123",
		Company:  "qtera mandiri",
	})
	if err != nil {
		t.Fatalf("unexpected registration error: %v", err)
	}
	if regRes.Token == "" {
		t.Fatal("expected non-empty JWT token")
	}
	if regRes.User.Username != "newuser" {
		t.Errorf("expected newuser, got %s", regRes.User.Username)
	}

	// 3. Login test with wrong password
	_, err = svc.Login(ctx, dto.LoginRequest{
		Username: "newuser",
		Password: "wrongpassword",
	})
	if err == nil {
		t.Fatal("expected error on invalid credentials, got nil")
	}

	// 4. Successful login
	loginRes, err := svc.Login(ctx, dto.LoginRequest{
		Username: "newuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected login error: %v", err)
	}
	if loginRes.Token == "" {
		t.Fatal("expected valid token upon successful login")
	}

	// 5. GetMe test
	meRes, err := svc.GetMe(ctx, loginRes.User.ID)
	if err != nil {
		t.Fatalf("unexpected GetMe error: %v", err)
	}
	if meRes.Email != "newuser@gmail.com" {
		t.Errorf("expected newuser@gmail.com, got %s", meRes.Email)
	}
}
