package models

// AttachmentInfo represents metadata for an attached file.
type AttachmentInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

// Chat maps to public.chat in PostgreSQL.
type Chat struct {
	ID             string          `json:"id"`
	IsUser         bool            `json:"user"` // maps to PG column "user"
	Chat           string          `json:"chat"`
	CreatedAt      string          `json:"created_at"`
	UserID         *int64          `json:"user_id,omitempty"`
	Username       string          `json:"username,omitempty"`
	Role           string          `json:"role,omitempty"`
	Models         string          `json:"models,omitempty"`
	AttachmentName *string         `json:"attachment_name,omitempty"`
	AttachmentURL  *string         `json:"attachment_url,omitempty"`
	AttachmentType *string         `json:"attachment_type,omitempty"`
	AttachmentSize *int64          `json:"attachment_size,omitempty"`
	AttachmentText *string         `json:"attachment_text,omitempty"`
	Attachment     *AttachmentInfo `json:"attachment,omitempty"`
}
