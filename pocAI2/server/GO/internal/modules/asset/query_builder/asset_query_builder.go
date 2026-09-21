package query_builder

import (
	"fmt"
	"strings"

	"golang-dh/internal/modules/asset/dto"
)

// BuildAssetFilterQuery constructs the WHERE clause, ORDER BY, and pagination for asset queries.
func BuildAssetFilterQuery(filter dto.AssetListFilter) (whereClause string, args []any, sortClause string, limitClause string) {
	var conditions []string
	idx := 1

	// Multi-tenant company isolation
	if filter.Perusahaan != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(TRIM(perusahaan)) = LOWER(TRIM($%d))", idx))
		args = append(args, filter.Perusahaan)
		idx++
	}

	// Search keyword across name, asset_id, category, brand
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR asset_id ILIKE $%d OR category ILIKE $%d OR brand ILIKE $%d)", idx, idx, idx, idx))
		args = append(args, "%"+filter.Search+"%")
		idx++
	}

	// Multi-value category filter
	if filter.Category != "" {
		cats := strings.Split(filter.Category, ",")
		var catParams []string
		for _, cat := range cats {
			cat = strings.TrimSpace(cat)
			if cat != "" {
				catParams = append(catParams, fmt.Sprintf("$%d", idx))
				args = append(args, cat)
				idx++
			}
		}
		if len(catParams) > 0 {
			conditions = append(conditions, "category IN ("+strings.Join(catParams, ",")+")")
		}
	}

	// Multi-value brand filter
	if filter.Brand != "" {
		brands := strings.Split(filter.Brand, ",")
		var brandParams []string
		for _, b := range brands {
			b = strings.TrimSpace(b)
			if b != "" {
				brandParams = append(brandParams, fmt.Sprintf("$%d", idx))
				args = append(args, b)
				idx++
			}
		}
		if len(brandParams) > 0 {
			conditions = append(conditions, "brand IN ("+strings.Join(brandParams, ",")+")")
		}
	}

	// Multi-value location filter
	if filter.Location != "" {
		locs := strings.Split(filter.Location, ",")
		var locParams []string
		for _, loc := range locs {
			loc = strings.TrimSpace(loc)
			if loc != "" {
				locParams = append(locParams, fmt.Sprintf("$%d", idx))
				args = append(args, loc)
				idx++
			}
		}
		if len(locParams) > 0 {
			conditions = append(conditions, "location IN ("+strings.Join(locParams, ",")+")")
		}
	}

	// Status filter
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status ILIKE $%d", idx))
		args = append(args, filter.Status)
		idx++
	}

	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Sort column mapping
	colMap := map[string]string{
		"createdAt":     "created_at",
		"assetId":       "asset_id",
		"name":          "name",
		"category":      "category",
		"brand":         "brand",
		"modelType":     "model_type",
		"purchaseDate":  "purchase_date",
		"purchasePrice": "purchase_price",
		"location":      "location",
		"status":        "status",
	}
	dbSortCol := "created_at"
	if col, ok := colMap[filter.Sort]; ok {
		dbSortCol = col
	}

	order := strings.ToUpper(filter.Order)
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}
	sortClause = fmt.Sprintf(" ORDER BY %s %s", dbSortCol, order)

	// Pagination
	if !filter.AllRecords && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		limitClause = fmt.Sprintf(" LIMIT %d OFFSET %d", filter.PageSize, offset)
	}

	return whereClause, args, sortClause, limitClause
}
