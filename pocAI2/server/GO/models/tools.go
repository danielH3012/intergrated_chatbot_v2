package models

import "time"

// ToolsHistoryPayload maps to public.tools_history in PostgreSQL.
type ToolsHistoryPayload struct {
	ID                string    `json:"id"`
	Perusahaan        string    `json:"perusahaan"`
	Tools             string    `json:"tools"`
	CreatedByUserID   string    `json:"created_by_user_id,omitempty"`
	CreatedByUsername string    `json:"created_by_username,omitempty"`
	CreatedByRole     string    `json:"created_by_role,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}
