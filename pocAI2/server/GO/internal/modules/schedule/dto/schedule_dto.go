package dto

// ScheduleItem represents an individual schedule record.
type ScheduleItem struct {
	IDSchedule string `json:"id_schedule"`
	Category   string `json:"category"`
	Start      string `json:"start"`
	Frequency  string `json:"frequency"`
}

// CreateScheduleRequest represents payload for creating a schedule.
type CreateScheduleRequest struct {
	Category  string `json:"category" form:"category"`
	Start     string `json:"start" form:"start"`
	Frequency string `json:"frequency" form:"frequency"`
}
