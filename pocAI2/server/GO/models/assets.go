package models

import (
	"time"
)

// Asset maps to public.assets in PostgreSQL.
type Asset struct {
	AssetID       string `json:"assetId"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	Brand         string `json:"brand"`
	ModelType     string `json:"modelType"`
	PurchaseDate  string `json:"purchaseDate"`
	PurchasePrice string `json:"purchasePrice"`
	Location      string `json:"location"`
	CreatedAt     string `json:"createdAt"`
	Perusahaan    string `json:"perusahaan"`
	Status        string `json:"status"`
}

// AIRegistrationSession maps to public.ai_registration_sessions in PostgreSQL.
type AIRegistrationSession struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	SourceFileURL string    `json:"sourceFileUrl"`
	RawText       string    `json:"rawText"`
	Status        string    `json:"status"`
	RejectReason  string    `json:"rejectReason,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

// AIRegistrationDraft maps to public.ai_registration_drafts in PostgreSQL.
type AIRegistrationDraft struct {
	ID             string                 `json:"id"`
	SessionID      string                 `json:"sessionId"`
	SequenceNo     int                    `json:"sequenceNo"`
	Fields         map[string]interface{} `json:"fields"`
	Status         string                 `json:"status"`
	CreatedAssetID string                 `json:"createdAssetId,omitempty"`
	LowConfidence  bool                   `json:"lowConfidence"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}
