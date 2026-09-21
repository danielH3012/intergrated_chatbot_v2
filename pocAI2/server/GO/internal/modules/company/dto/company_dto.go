package dto

// CompanyItem represents an individual company record.
type CompanyItem struct {
	ID   int64  `json:"id_perusahaan"`
	Name string `json:"nama_perusahaan"`
}

// CompanyListResponse represents the list of companies.
type CompanyListResponse struct {
	Companies []CompanyItem `json:"companies"`
}
