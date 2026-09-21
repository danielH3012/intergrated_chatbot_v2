package dto

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
}

// AssetListFilter represents query criteria for filtering assets.
type AssetListFilter struct {
	Perusahaan string
	Search     string
	Category   string
	Brand      string
	Location   string
	Status     string
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
	Name          string `json:"name"`
	Category      string `json:"category"`
	Brand         string `json:"brand"`
	ModelType     string `json:"modelType"`
	PurchaseDate  string `json:"purchaseDate"`
	PurchasePrice string `json:"purchasePrice"`
	Location      string `json:"location"`
	Perusahaan    string `json:"perusahaan"`
	Status        string `json:"status"`
}

// UpdateAssetRequest represents fields allowed during partial update.
type UpdateAssetRequest struct {
	Name          *string `json:"name"`
	Category      *string `json:"category"`
	Brand         *string `json:"brand"`
	ModelType     *string `json:"modelType"`
	PurchaseDate  *string `json:"purchaseDate"`
	PurchasePrice *string `json:"purchasePrice"`
	Location      *string `json:"location"`
	Status        *string `json:"status"`
}
