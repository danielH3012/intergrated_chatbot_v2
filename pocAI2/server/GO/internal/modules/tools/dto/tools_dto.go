package dto

import "time"

// CreateToolsHistoryRequest represents payload for recording tool executions.
type CreateToolsHistoryRequest struct {
	Perusahaan        string `json:"perusahaan"`
	Tools             string `json:"tools"`
	CreatedByUserID   string `json:"created_by_user_id"`
	CreatedByUsername string `json:"created_by_username"`
	CreatedByRole     string `json:"created_by_role"`
}

// ToolsHistoryItem represents an audit ledger entry.
type ToolsHistoryItem struct {
	ID                string    `json:"id"`
	Perusahaan        string    `json:"perusahaan"`
	Tools             string    `json:"tools"`
	CreatedByUserID   string    `json:"created_by_user_id"`
	CreatedByUsername string    `json:"created_by_username"`
	CreatedByRole     string    `json:"created_by_role"`
	CreatedAt         time.Time `json:"created_at"`
}
