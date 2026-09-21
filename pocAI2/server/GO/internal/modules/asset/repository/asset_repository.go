package repository

import (
	"context"
	"fmt"
	"strings"

	"golang-dh/internal/modules/asset/dto"
	"golang-dh/internal/modules/asset/query_builder"
	"golang-dh/pkg/database"
)

// AssetRepository defines data access methods for IT assets.
type AssetRepository interface {
	List(ctx context.Context, filter dto.AssetListFilter) ([]dto.AssetItem, int, error)
	GetByID(ctx context.Context, assetID string) (*dto.AssetItem, error)
	GetOptions(ctx context.Context, perusahaan string) (dto.AssetOptionsResponse, error)
	Create(ctx context.Context, asset dto.AssetItem) error
	Update(ctx context.Context, assetID string, fields map[string]any) error
	Delete(ctx context.Context, assetID string) error
	Exists(ctx context.Context, assetID string) (bool, error)
}

type assetRepository struct {
	db database.PostgreSQLServicer
}

// NewAssetRepository creates a new AssetRepository.
func NewAssetRepository(db database.PostgreSQLServicer) AssetRepository {
	return &assetRepository{db: db}
}

func (r *assetRepository) List(ctx context.Context, filter dto.AssetListFilter) ([]dto.AssetItem, int, error) {
	whereClause, args, sortClause, limitClause := query_builder.BuildAssetFilterQuery(filter)

	// Count total matching items
	countQuery := "SELECT COUNT(*) FROM public.assets" + whereClause
	var totalItems int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, 0, err
	}

	// Fetch data rows
	dataQuery := "SELECT asset_id, name, category, brand, model_type, purchase_date, purchase_price, location, created_at, perusahaan, status FROM public.assets" + whereClause + sortClause + limitClause
	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assets []dto.AssetItem
	for rows.Next() {
		var a dto.AssetItem
		if err := rows.Scan(&a.AssetID, &a.Name, &a.Category, &a.Brand, &a.ModelType, &a.PurchaseDate, &a.PurchasePrice, &a.Location, &a.CreatedAt, &a.Perusahaan, &a.Status); err != nil {
			return nil, 0, err
		}
		assets = append(assets, a)
	}
	if assets == nil {
		assets = []dto.AssetItem{}
	}

	return assets, totalItems, nil
}

func (r *assetRepository) GetByID(ctx context.Context, assetID string) (*dto.AssetItem, error) {
	query := `SELECT asset_id, name, category, brand, model_type, purchase_date, purchase_price, location, created_at, perusahaan, status FROM public.assets WHERE asset_id ILIKE $1 LIMIT 1`
	var a dto.AssetItem
	err := r.db.QueryRow(ctx, query, assetID).Scan(&a.AssetID, &a.Name, &a.Category, &a.Brand, &a.ModelType, &a.PurchaseDate, &a.PurchasePrice, &a.Location, &a.CreatedAt, &a.Perusahaan, &a.Status)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *assetRepository) GetOptions(ctx context.Context, perusahaan string) (dto.AssetOptionsResponse, error) {
	var whereClause string
	var args []any
	if perusahaan != "" {
		whereClause = " WHERE LOWER(TRIM(perusahaan)) = LOWER(TRIM($1))"
		args = append(args, perusahaan)
	}

	fetchDistinct := func(column string) []string {
		q := fmt.Sprintf("SELECT DISTINCT %s FROM public.assets%s ORDER BY %s", column, whereClause, column)
		rows, err := r.db.Query(ctx, q, args...)
		if err != nil {
			return []string{}
		}
		defer rows.Close()

		var results []string
		for rows.Next() {
			var val *string
			if err := rows.Scan(&val); err == nil && val != nil && strings.TrimSpace(*val) != "" {
				results = append(results, *val)
			}
		}
		if results == nil {
			results = []string{}
		}
		return results
	}

	return dto.AssetOptionsResponse{
		Categories: fetchDistinct("category"),
		Locations:  fetchDistinct("location"),
		Brands:     fetchDistinct("brand"),
	}, nil
}

func (r *assetRepository) Create(ctx context.Context, a dto.AssetItem) error {
	query := `
		INSERT INTO public.assets (asset_id, name, category, brand, model_type, purchase_date, purchase_price, location, created_at, perusahaan, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Exec(ctx, query, a.AssetID, a.Name, a.Category, a.Brand, a.ModelType, a.PurchaseDate, a.PurchasePrice, a.Location, a.CreatedAt, a.Perusahaan, a.Status)
	return err
}

func (r *assetRepository) Update(ctx context.Context, assetID string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}

	var setClauses []string
	var args []any
	idx := 1

	for col, val := range fields {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, val)
		idx++
	}

	query := fmt.Sprintf("UPDATE public.assets SET %s WHERE asset_id = $%d", strings.Join(setClauses, ", "), idx)
	args = append(args, assetID)

	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *assetRepository) Delete(ctx context.Context, assetID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM public.assets WHERE asset_id ILIKE $1`, assetID)
	return err
}

func (r *assetRepository) Exists(ctx context.Context, assetID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM public.assets WHERE asset_id ILIKE $1`, assetID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
