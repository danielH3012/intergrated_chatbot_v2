package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"golang-dh/internal/modules/schedule/dto"
)

// ScheduleRepositoryFake provides an in-memory repository for schedule testing.
type ScheduleRepositoryFake struct {
	Items []dto.ScheduleItem
}

func NewScheduleRepositoryFake(initial []dto.ScheduleItem) *ScheduleRepositoryFake {
	return &ScheduleRepositoryFake{Items: initial}
}

func (f *ScheduleRepositoryFake) EnsureTable(ctx context.Context) error {
	return nil
}

func (f *ScheduleRepositoryFake) Create(ctx context.Context, category, start, frequency string) (string, error) {
	newID := uuid.New().String()
	f.Items = append(f.Items, dto.ScheduleItem{
		IDSchedule: newID,
		Category:   category,
		Start:      start,
		Frequency:  frequency,
	})
	return newID, nil
}

func (f *ScheduleRepositoryFake) List(ctx context.Context, categoryFilter string) ([]dto.ScheduleItem, error) {
	if strings.TrimSpace(categoryFilter) == "" {
		return f.Items, nil
	}
	var res []dto.ScheduleItem
	for _, it := range f.Items {
		if strings.Contains(strings.ToLower(it.Category), strings.ToLower(categoryFilter)) {
			res = append(res, it)
		}
	}
	return res, nil
}

func (f *ScheduleRepositoryFake) Delete(ctx context.Context, id string) (int64, error) {
	var remaining []dto.ScheduleItem
	var deleted int64 = 0
	for _, it := range f.Items {
		if it.IDSchedule == id {
			deleted++
		} else {
			remaining = append(remaining, it)
		}
	}
	f.Items = remaining
	return deleted, nil
}
