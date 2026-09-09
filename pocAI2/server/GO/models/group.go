package models

import "time"

// GroupPayload maps to public.groups in PostgreSQL.
// PG column "group_name" was "group" in the legacy MongoDB collection.
type GroupPayload struct {
	ID                string    `json:"id"`
	Perusahaan        string    `json:"perusahaan"`
	GroupName         string    `json:"group_name"`       // PG col: group_name
	Application       string    `json:"application,omitempty"`
	AgentID           string    `json:"agent_id,omitempty"`
	CreatedByUserID   string    `json:"created_by_user_id,omitempty"`
	CreatedByUsername string    `json:"created_by_username,omitempty"`
	CreatedByRole     string    `json:"created_by_role,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}
