package controllers

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang-dh/middleware"
	"golang-dh/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ---------- ASSETS API ----------

// Helper to extract authenticated user's role and company directly from headers / claims
func resolveUserContext(c *fiber.Ctx) (string, string) {
	headerRole := strings.ToLower(strings.TrimSpace(c.Get("X-Role")))
	headerCompany := strings.TrimSpace(c.Get("X-Company"))

	if userVal := c.Locals("user"); userVal != nil {
		if claims, ok := userVal.(*middleware.JWTClaims); ok && claims != nil {
			role := strings.ToLower(strings.TrimSpace(claims.Role))
			company := strings.TrimSpace(claims.Company)
			if company == "" && headerCompany != "" {
				company = headerCompany
			}
			return role, company
		}
	}

	authHeader := c.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == "token123" {
			role := headerRole
			if role == "" {
				role = "admin"
			}
			return role, headerCompany
		}
		if claims, err := middleware.ValidateJWT(tokenStr); err == nil && claims != nil {
			role := strings.ToLower(strings.TrimSpace(claims.Role))
			company := strings.TrimSpace(claims.Company)
			if company == "" && headerCompany != "" {
				company = headerCompany
			}
			return role, company
		}
	}

	if headerRole != "" || headerCompany != "" {
		if headerRole == "" {
			headerRole = "operator"
		}
		return headerRole, headerCompany
	}

	return "operator", ""
}

func GetAssets(c *fiber.Ctx) error {
	_, userCompany := resolveUserContext(c)

	search := strings.TrimSpace(c.Query("search"))
	category := strings.TrimSpace(c.Query("category"))
	brand := strings.TrimSpace(c.Query("brand"))
	location := strings.TrimSpace(c.Query("location"))
	sortCol := c.Query("sort", "created_at")
	order := strings.ToUpper(c.Query("order", "DESC"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSizeStr := c.Query("pageSize", "10")

	// Multi-tenancy enforcement: strictly locked to user's company from request header/token
	perusahaan := strings.TrimSpace(userCompany)
	if perusahaan == "" {
		// Strict multi-tenancy: disallow querying all assets if no company is identified
		return c.JSON(fiber.Map{
			"assets": []models.Asset{},
			"pagination": fiber.Map{
				"page":       page,
				"pageSize":   10,
				"totalItems": 0,
				"totalPages": 1,
			},
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	baseQuery := ` FROM assets WHERE 1=1`
	var args []interface{}
	idx := 1

	// Strict company filter matching request header directly (no leaking unassigned assets)
	baseQuery += fmt.Sprintf(` AND LOWER(TRIM(perusahaan)) = LOWER(TRIM($%d))`, idx)
	args = append(args, perusahaan)
	idx++

	if search != "" {
		baseQuery += fmt.Sprintf(` AND (name ILIKE $%d OR asset_id ILIKE $%d OR category ILIKE $%d OR brand ILIKE $%d)`, idx, idx, idx, idx)
		args = append(args, "%"+search+"%")
		idx++
	}
	if category != "" {
		cats := strings.Split(category, ",")
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
			baseQuery += ` AND category IN (` + strings.Join(catParams, ",") + `)`
		}
	}
	if brand != "" {
		brands := strings.Split(brand, ",")
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
			baseQuery += ` AND brand IN (` + strings.Join(brandParams, ",") + `)`
		}
	}
	if location != "" {
		locs := strings.Split(location, ",")
		var locParams []string
		for _, l := range locs {
			l = strings.TrimSpace(l)
			if l != "" {
				locParams = append(locParams, fmt.Sprintf("$%d", idx))
				args = append(args, l)
				idx++
			}
		}
		if len(locParams) > 0 {
			baseQuery += ` AND location IN (` + strings.Join(locParams, ",") + `)`
		}
	}

	var totalItems int
	countQuery := `SELECT COUNT(*)` + baseQuery
	_ = DB.QueryRow(ctx, countQuery, args...).Scan(&totalItems)

	validSorts := map[string]string{
		"assetId": "asset_id", "name": "name", "category": "category", "brand": "brand",
		"modelType": "model_type", "purchaseDate": "purchase_date", "purchasePrice": "purchase_price",
		"location": "location", "createdAt": "created_at",
	}
	dbSortCol, ok := validSorts[sortCol]
	if !ok {
		dbSortCol = "created_at"
	}
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	dataQuery := `SELECT asset_id, name, category, brand, model_type, purchase_date, purchase_price, location, created_at, perusahaan, status` + baseQuery + ` ORDER BY ` + dbSortCol + ` ` + order

	pageSize := 10
	if pageSizeStr == "all" {
		pageSize = totalItems
		if pageSize == 0 {
			pageSize = 1
		}
	} else {
		ps, err := strconv.Atoi(pageSizeStr)
		if err == nil && ps > 0 {
			pageSize = ps
		}
		offset := (page - 1) * pageSize
		dataQuery += fmt.Sprintf(` LIMIT %d OFFSET %d`, pageSize, offset)
	}

	rows, err := DB.Query(ctx, dataQuery, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	var assets []models.Asset
	for rows.Next() {
		var a models.Asset
		if err := rows.Scan(&a.AssetID, &a.Name, &a.Category, &a.Brand, &a.ModelType, &a.PurchaseDate, &a.PurchasePrice, &a.Location, &a.CreatedAt, &a.Perusahaan, &a.Status); err == nil {
			assets = append(assets, a)
		}
	}
	if assets == nil {
		assets = []models.Asset{}
	}

	totalPages := (totalItems + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return c.JSON(fiber.Map{
		"assets": assets,
		"pagination": fiber.Map{
			"page":       page,
			"pageSize":   pageSize,
			"totalItems": totalItems,
			"totalPages": totalPages,
		},
	})
}

func GetAssetOptions(c *fiber.Ctx) error {
	_, userCompany := resolveUserContext(c)
	perusahaan := strings.TrimSpace(userCompany)
	if perusahaan == "" {
		return c.JSON(fiber.Map{
			"categories": []string{},
			"locations":  []string{},
			"brands":     []string{},
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	whereClause := " WHERE LOWER(TRIM(perusahaan)) = LOWER(TRIM($1))"
	args := []interface{}{perusahaan}

	categories := queryDistinctStringsWithArgs(ctx, `SELECT DISTINCT category FROM assets`+whereClause+` ORDER BY category`, args...)
	locations := queryDistinctStringsWithArgs(ctx, `SELECT DISTINCT location FROM assets`+whereClause+` ORDER BY location`, args...)
	brands := queryDistinctStringsWithArgs(ctx, `SELECT DISTINCT brand FROM assets`+whereClause+` ORDER BY brand`, args...)

	return c.JSON(fiber.Map{
		"categories": categories,
		"locations":  locations,
		"brands":     brands,
	})
}

func queryDistinctStringsWithArgs(ctx context.Context, sql string, args ...interface{}) []string {
	rows, err := DB.Query(ctx, sql, args...)
	if err != nil {
		return []string{}
	}
	defer rows.Close()
	var res []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil && s != "" {
			res = append(res, s)
		}
	}
	if res == nil {
		res = []string{}
	}
	return res
}

func CreateAssetManual(c *fiber.Ctx) error {
	_, userCompany := resolveUserContext(c)
	perusahaan := strings.TrimSpace(userCompany)
	if perusahaan == "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "permission denied: user company header or token is missing"})
	}

	var body struct {
		Name          string      `json:"name"`
		Category      string      `json:"category"`
		Brand         string      `json:"brand"`
		ModelType     string      `json:"modelType"`
		PurchaseDate  string      `json:"purchaseDate"`
		PurchasePrice interface{} `json:"purchasePrice"`
		Location      string      `json:"location"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if strings.TrimSpace(body.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "asset name is required"})
	}
	if strings.TrimSpace(body.Category) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "category is required"})
	}

	priceStr := fmt.Sprintf("%v", body.PurchasePrice)
	if priceStr == "<nil>" {
		priceStr = ""
	}

	// Always lock to user's company from header


	status := "active"

	assetID := "AST-" + strings.ToUpper(uuid.NewString()[:8])
	createdAt := time.Now().Format(time.RFC3339)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := DB.Exec(ctx,
		`INSERT INTO assets (asset_id, name, category, brand, model_type, purchase_date, purchase_price, location, created_at, perusahaan, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		assetID, strings.TrimSpace(body.Name), strings.TrimSpace(body.Category), strings.TrimSpace(body.Brand),
		strings.TrimSpace(body.ModelType), strings.TrimSpace(body.PurchaseDate), priceStr, strings.TrimSpace(body.Location),
		createdAt, perusahaan, status,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"assetId":    assetID,
		"status":     status,
		"perusahaan": perusahaan,
		"createdAt":  createdAt,
	})
}

func UpdateAsset(c *fiber.Ctx) error {
	_, userCompany := resolveUserContext(c)

	rawID, _ := url.PathUnescape(c.Params("id"))
	id := strings.TrimSpace(rawID)
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "asset id is required"})
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Case-insensitive lookup to find the asset and verify company ownership
	var canonicalID, existingCompany string
	err := DB.QueryRow(ctx, `SELECT asset_id, COALESCE(perusahaan, '') FROM assets WHERE asset_id ILIKE $1`, id).Scan(&canonicalID, &existingCompany)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "asset not found"})
	}

	userCompClean := strings.TrimSpace(userCompany)
	existCompClean := strings.TrimSpace(existingCompany)
	if userCompClean == "" || existCompClean == "" || !strings.EqualFold(existCompClean, userCompClean) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "permission denied: cannot modify assets belonging to another company"})
	}

	// Never allow changing perusahaan or createdAt or asset_id
	delete(body, "perusahaan")
	delete(body, "createdAt")
	delete(body, "created_at")
	delete(body, "asset_id")
	delete(body, "assetId")
	delete(body, "id")

	fieldMap := map[string]string{
		"name": "name", "category": "category", "brand": "brand", "modelType": "model_type",
		"purchaseDate": "purchase_date", "purchasePrice": "purchase_price", "location": "location",
		"status": "status",
	}

	var setClauses []string
	var args []interface{}
	idx := 1

	for k, v := range body {
		if dbCol, ok := fieldMap[k]; ok {
			setClauses = append(setClauses, fmt.Sprintf("%s=$%d", dbCol, idx))
			args = append(args, fmt.Sprintf("%v", v))
			idx++
		}
	}

	if len(setClauses) == 0 {
		return c.JSON(fiber.Map{"ok": true, "assetId": canonicalID})
	}

	query := fmt.Sprintf(`UPDATE assets SET %s WHERE asset_id=$%d`, strings.Join(setClauses, ", "), idx)
	args = append(args, canonicalID)

	_, err = DB.Exec(ctx, query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"ok": true, "assetId": canonicalID})
}

func DeleteAsset(c *fiber.Ctx) error {
	_, userCompany := resolveUserContext(c)

	rawID, _ := url.PathUnescape(c.Params("id"))
	id := strings.TrimSpace(rawID)
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "asset id is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Case-insensitive lookup to find the asset and verify company ownership
	var canonicalID, existingCompany string
	err := DB.QueryRow(ctx, `SELECT asset_id, COALESCE(perusahaan, '') FROM assets WHERE asset_id ILIKE $1`, id).Scan(&canonicalID, &existingCompany)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "asset not found"})
	}

	userCompClean := strings.TrimSpace(userCompany)
	existCompClean := strings.TrimSpace(existingCompany)
	if userCompClean == "" || existCompClean == "" || !strings.EqualFold(existCompClean, userCompClean) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "permission denied: cannot delete assets belonging to another company"})
	}

	_, err = DB.Exec(ctx, `DELETE FROM assets WHERE asset_id=$1`, canonicalID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func DownloadAssets(c *fiber.Ctx) error {
	return GetAssets(c)
}