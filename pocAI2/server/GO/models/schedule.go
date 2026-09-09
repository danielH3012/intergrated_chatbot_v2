package models

// Schedule represents a record in public.schedule.
type Schedule struct {
	IDSchedule string `json:"id_schedule"`
	Category   string `json:"category"`
	Start      string `json:"start"`
	Frequency  string `json:"frequency"`
}
