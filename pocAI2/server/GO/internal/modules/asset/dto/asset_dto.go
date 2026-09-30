package dto

import (
	"encoding/json"
	"strings"
)

// FlexibleString allows unmarshaling JSON strings, numbers, or null into a Go string.
type FlexibleString string

func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	str := string(data)
	if str == "null" || str == `""` {
		*fs = ""
		return nil
	}
	if strings.HasPrefix(str, `"`) && strings.HasSuffix(str, `"`) {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*fs = FlexibleString(s)
		return nil
	}
	*fs = FlexibleString(strings.Trim(str, `"`))
	return nil
}

func (fs FlexibleString) String() string {
	return string(fs)
}

// AssetItem represents an individual IT asset entity.
type AssetItem struct {
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
	Condition     string `json:"condition"`
}

// AssetListFilter represents query criteria for filtering assets.
type AssetListFilter struct {
	Perusahaan string
	Search     string
	Category   string
	Brand      string
	Location   string
	Status     string
	Condition  string
	Sort       string
	Order      string
	Page       int
	PageSize   int
	AllRecords bool
}

// AssetListResult holds paginated results.
type AssetListResult struct {
	Assets     []AssetItem `json:"assets"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalItems int         `json:"totalItems"`
	TotalPages int         `json:"totalPages"`
}

// AssetOptionsResponse contains distinct filter values for the UI.
type AssetOptionsResponse struct {
	Categories []string `json:"categories"`
	Locations  []string `json:"locations"`
	Brands     []string `json:"brands"`
}

// CreateAssetRequest represents payload for creating an asset manually.
type CreateAssetRequest struct {
	Name          string         `json:"name"`
	Category      string         `json:"category"`
	Brand         string         `json:"brand"`
	ModelType     string         `json:"modelType"`
	PurchaseDate  string         `json:"purchaseDate"`
	PurchasePrice FlexibleString `json:"purchasePrice"`
	Location      string         `json:"location"`
	Perusahaan    string         `json:"perusahaan"`
	Status        string         `json:"status"`
	Condition     string         `json:"condition"`
}

// UpdateAssetRequest represents fields allowed during partial update.
type UpdateAssetRequest struct {
	Name          *string         `json:"name"`
	Category      *string         `json:"category"`
	Brand         *string         `json:"brand"`
	ModelType     *string         `json:"modelType"`
	PurchaseDate  *string         `json:"purchaseDate"`
	PurchasePrice *FlexibleString `json:"purchasePrice"`
	Location      *string         `json:"location"`
	Status        *string         `json:"status"`
	Condition     *string         `json:"condition"`
}
