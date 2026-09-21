package service_test

import (
	"context"
	"testing"

	"golang-dh/internal/modules/schedule/dto"
	"golang-dh/internal/modules/schedule/repository"
	"golang-dh/internal/modules/schedule/service"
)

func TestScheduleService_CreateAndList(t *testing.T) {
	fakeRepo := repository.NewScheduleRepositoryFake(nil)
	svc := service.NewScheduleService(fakeRepo)
	ctx := context.Background()

	// 1. Test validation
	_, err := svc.CreateSchedule(ctx, dto.CreateScheduleRequest{
		Category: "",
	})
	if err == nil {
		t.Fatal("expected error on empty category, got nil")
	}

	// 2. Test successful creation
	id, err := svc.CreateSchedule(ctx, dto.CreateScheduleRequest{
		Category:  "Pemeriksaan Aset",
		Start:     "2026-10-01",
		Frequency: "Weekly",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty schedule ID")
	}

	// 3. Test listing
	schedules, err := svc.GetSchedules(ctx, "")
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(schedules))
	}

	// 4. Test delete
	err = svc.DeleteSchedule(ctx, id)
	if err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}

	schedulesAfter, _ := svc.GetSchedules(ctx, "")
	if len(schedulesAfter) != 0 {
		t.Fatalf("expected 0 schedules after delete, got %d", len(schedulesAfter))
	}
}
