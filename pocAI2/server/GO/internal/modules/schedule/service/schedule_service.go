package service

import (
	"context"
	"errors"
	"strings"

	"golang-dh/internal/modules/schedule/dto"
	"golang-dh/internal/modules/schedule/repository"
)

// ScheduleService handles business logic for schedule management.
type ScheduleService struct {
	repo repository.ScheduleRepository
}

// NewScheduleService creates a new ScheduleService.
func NewScheduleService(repo repository.ScheduleRepository) *ScheduleService {
	return &ScheduleService{repo: repo}
}

func (s *ScheduleService) EnsureTable(ctx context.Context) error {
	return s.repo.EnsureTable(ctx)
}

func (s *ScheduleService) CreateSchedule(ctx context.Context, req dto.CreateScheduleRequest) (string, error) {
	category := strings.TrimSpace(req.Category)
	start := strings.TrimSpace(req.Start)
	frequency := strings.TrimSpace(req.Frequency)

	if category == "" {
		return "", errors.New("category is required")
	}
	if start == "" {
		return "", errors.New("start date/time is required")
	}
	if frequency == "" {
		return "", errors.New("frequency is required")
	}

	return s.repo.Create(ctx, category, start, frequency)
}

func (s *ScheduleService) GetSchedules(ctx context.Context, categoryFilter string) ([]dto.ScheduleItem, error) {
	return s.repo.List(ctx, categoryFilter)
}

func (s *ScheduleService) DeleteSchedule(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return errors.New("id is required")
	}
	rowsAffected, err := s.repo.Delete(ctx, trimmedID)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("schedule not found or already deleted")
	}
	return nil
}
