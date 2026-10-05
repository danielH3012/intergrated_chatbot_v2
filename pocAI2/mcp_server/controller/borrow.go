package controller

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp_server/client"
)

// RegisterBorrowTools mounts borrow transaction tools onto the MCP server.
func RegisterBorrowTools(s *server.MCPServer) {
	// 1. create_borrow_transaction: submit a new borrow request into hold_borrow queue
	// borrower_name is set from AI input, while manager_name & manager_id are taken from user credentials.
	// All borrow requests created by AI are placed in the hold_borrow table until manually confirmed/approved by user.
	s.AddTool(mcp.NewTool("create_borrow_transaction",
		mcp.WithDescription(
			`Create an asset borrow request in the system hold queue.
All borrow requests created by the AI are held in the 'hold_borrow' table (status: held) before being manually reviewed and submitted by the user on the Hold Borrow page.
Use this tool whenever the user asks to borrow an asset, create a borrow request, submit a loan transaction, or record equipment borrowing (e.g. 'pinjam laptop Dell selama 7 hari', 'buat transaksi peminjaman aset AST-6D2933A2 untuk Budi', 'ajukan peminjaman monitor 14 hari peminjam Dan').
Parameters:
  - asset_id: Target unique asset identifier (e.g. 'AST-XXXXXXXX')
  - borrower_name: Name of the employee or person borrowing the asset (e.g. 'Budi Santoso', 'jason')
  - duration_days: Duration of the borrow period in days (integer, e.g. 7 or 14, defaults to 14)
  - group_name: Borrower's group, division, or department (defaults to user's company or 'qtera mandiri')
  - items: Optional list of multiple items if borrowing multiple assets at once`),
		mcp.WithString("asset_id", mcp.Description("Target unique asset identifier (e.g. 'AST-XXXXXXXX') or comma-separated DISTINCT IDs")),
		mcp.WithString("borrower_name", mcp.Description("Name of the employee or borrower (e.g. 'Budi Santoso', 'jason')")),
		mcp.WithNumber("duration_days", mcp.Description("Borrow duration in days (e.g. 7, 14)")),
		mcp.WithString("group_name", mcp.Description("Borrower's group or department name")),
		mcp.WithNumber("quantity", mcp.Description("User-specified amount; must equal number of distinct IDs in items/asset_id (strict, no duplicates)")),
		mcp.WithArray("items", mcp.Description("Optional array of DISTINCT items: [{'asset_id': 'AST-...', 'duration_days': 14}]")),
	), HandleCreateBorrowTransaction)

	// 2. get_borrow_transactions: list or view borrow transactions
	s.AddTool(mcp.NewTool("get_borrow_transactions",
		mcp.WithDescription(
			`List or view existing borrow transactions and approval requests from the system.
Use this tool whenever the user asks to check, view, or list borrow transactions, approval statuses, or loans (e.g. 'tampilkan transaksi peminjaman', 'lihat daftar approval peminjaman', 'cek transaksi TRX-BRW-001').
Parameters:
  - id: Optional transaction ID or transaction code (e.g. 'TRX-BRW-001') to get specific transaction details`),
		mcp.WithString("id", mcp.Description("Optional transaction ID or transaction code (e.g. 'TRX-BRW-001')")),
	), HandleGetBorrowTransactions)

	// 2b. get_borrow_history_recommendations: get asset recommendations based on historical borrow patterns
	s.AddTool(mcp.NewTool("get_borrow_history_recommendations",
		mcp.WithDescription(
			`Get asset recommendations based on historical borrow transactions and loan patterns (most frequently borrowed or commonly used assets).
Use this tool whenever the user asks for recommendations of assets that are usually or commonly borrowed, popular, or frequently used for a department, division, or location (e.g. 'saya mau pinjam kursi untuk gudang apa rekomendasi yang biasa dipakai', 'laptop apa yang sering dipinjam', 'rekomendasi aset yang biasa digunakan', 'barang apa yang paling sering dipinjam').
Parameters:
  - query: Keyword or asset type (e.g. 'kursi', 'laptop', 'monitor')
  - category: Optional asset category (e.g. 'Furniture', 'IT Equipment')
  - location: Optional location or department (e.g. 'gudang', 'serpong')
  - group_name: Optional tenant / company name
  - limit: Maximum number of recommendations to return (integer, default 5)`),
		mcp.WithString("query", mcp.Description("Keyword or asset name/type to search (e.g. 'kursi', 'laptop', 'monitor')")),
		mcp.WithString("category", mcp.Description("Asset category filter (e.g. 'Furniture', 'IT Equipment')")),
		mcp.WithString("location", mcp.Description("Location or department filter (e.g. 'gudang', 'serpong')")),
		mcp.WithString("group_name", mcp.Description("Optional tenant / company name")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of recommendations (default 5)")),
	), HandleGetBorrowHistoryRecommendations)

	// 3. check_borrow_anomalies: inspect anomaly signals and recommendations for a transaction
	s.AddTool(mcp.NewTool("check_borrow_anomalies",
		mcp.WithDescription(
			`Inspect detected anomaly signals, behavioral risk context, and approval recommendation for a specific borrow transaction.
Use this tool whenever the user asks to check anomalies, fraud risks, or compliance analysis for a transaction (e.g. 'cek anomali transaksi TRX-BRW-001', 'apakah transaksi TRX-BRW-001 ada anomali?').
Parameters:
  - id: Target transaction ID or transaction code (e.g. 'TRX-BRW-001')`),
		mcp.WithString("id", mcp.Required(), mcp.Description("Target transaction ID or code (e.g. 'TRX-BRW-001')")),
	), HandleCheckBorrowAnomalies)

	// 4. get_borrow_threshold_settings: view borrow limits and anomaly thresholds
	s.AddTool(mcp.NewTool("get_borrow_threshold_settings",
		mcp.WithDescription(
			`View active borrow loan rules, duration limits, and anomaly detection thresholds for the current tenant/company.
Use this tool whenever the user asks to view, check, or inspect borrow threshold settings or loan rules (e.g. 'lihat threshold peminjaman', 'berapa batas durasi pinjam?', 'tampilkan aturan peminjaman').`),
		mcp.WithString("group_name", mcp.Description("Optional tenant / company name (defaults to user's company)")),
	), HandleGetBorrowThresholdSettings)

	// 5. update_borrow_threshold_settings: configure borrow limits and anomaly thresholds (ADMIN ONLY)
	s.AddTool(mcp.NewTool("update_borrow_threshold_settings",
		mcp.WithDescription(
			`Configure and update loan limits and anomaly detection thresholds for borrow transactions.
RESTRICTED: This action can ONLY be performed by users with the 'admin' role. Non-admin users are strictly denied.
Use this tool whenever an ADMINISTRATOR asks to set, update, or change borrow limits or anomaly thresholds (e.g. 'ubah batas durasi pinjam jadi 14 hari', 'set maksimal pinjaman aktif ke 5', 'atur batas keterlambatan pengembalian 3 kali').
Parameters:
  - max_duration_days: Maximum allowed borrow duration in days (integer, e.g. 14, 28)
  - max_active_loans: Maximum concurrent active loans allowed per employee (integer, e.g. 3, 5)
  - max_late_returns: Late return threshold count within 6 months (integer, e.g. 2)
  - max_extensions: Maximum loan extension count within 6 months (integer, e.g. 2)
  - max_asset_count: Maximum asset count per single borrow request (integer, e.g. 5)
  - audit_window_days: Days before stock-take/audit window to warn loans (integer, e.g. 7)
  - repeat_borrow_cycles: Repeat borrow cycles threshold of identical asset (integer, e.g. 2)
  - min_signals_for_review: Minimum triggered anomaly signals required to flag for review (integer, e.g. 1)
  - group_name: Optional target tenant / company name`),
		mcp.WithNumber("max_duration_days", mcp.Description("Maximum allowed borrow duration in days (e.g. 14, 28)")),
		mcp.WithNumber("max_active_loans", mcp.Description("Maximum concurrent active loans allowed per employee (e.g. 3, 5)")),
		mcp.WithNumber("max_late_returns", mcp.Description("Late return count threshold within 6 months (e.g. 2)")),
		mcp.WithNumber("max_extensions", mcp.Description("Maximum extension count threshold within 6 months (e.g. 2)")),
		mcp.WithNumber("max_asset_count", mcp.Description("Maximum asset items allowed in 1 request (e.g. 5)")),
		mcp.WithNumber("audit_window_days", mcp.Description("Audit window warning threshold in days (e.g. 7)")),
		mcp.WithNumber("repeat_borrow_cycles", mcp.Description("Repeat borrow cycles threshold for same asset (e.g. 2)")),
		mcp.WithNumber("min_signals_for_review", mcp.Description("Minimum triggered signals for review (e.g. 1)")),
		mcp.WithString("group_name", mcp.Description("Optional tenant / company name")),
	), HandleUpdateBorrowThresholdSettings)
}

// HandleCreateBorrowTransaction processes the creation of an asset borrow request.
func HandleCreateBorrowTransaction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)

	assetID := client.GetArgString(req, "asset_id", "asset", "id")
	borrowerName := client.GetArgString(req, "borrower_name", "peminjam", "nama_peminjam", "borrower", "name")
	borrowerID := client.GetArgString(req, "borrower_id", "user_id")
	groupName := client.GetArgString(req, "group_name", "group", "divisi", "department", "company")

	// manager_name and manager_id are taken from user credentials
	managerID := ""
	managerName := ""
	if creds != nil {
		if creds.NormalizedUserID() != "" {
			managerID = creds.NormalizedUserID()
		} else if creds.NormalizedName() != "" {
			managerID = creds.NormalizedName()
		}
		if creds.NormalizedName() != "" {
			managerName = creds.NormalizedName()
		} else if creds.NormalizedUserID() != "" {
			managerName = creds.NormalizedUserID()
		}
	}
	if managerID == "" || managerID == "-" || strings.EqualFold(managerID, "user") {
		managerID = "user"
	}
	if managerName == "" || managerName == "-" {
		managerName = managerID
	}

	durationDays := 14
	if durVal, ok := req.GetArguments()["duration_days"]; ok && durVal != nil {
		switch v := durVal.(type) {
		case float64:
			durationDays = int(v)
		case int:
			durationDays = v
		case string:
			if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && parsed > 0 {
				durationDays = parsed
			}
		}
	} else if durStr := client.GetArgString(req, "durasi", "duration", "days", "hari"); durStr != "" {
		if parsed, err := strconv.Atoi(durStr); err == nil && parsed > 0 {
			durationDays = parsed
		}
	}

	if durationDays <= 0 {
		durationDays = 14
	}

	topQuantity := 1
	quantityExplicit := false
	if qVal, ok := req.GetArguments()["quantity"]; ok && qVal != nil {
		switch q := qVal.(type) {
		case float64:
			if int(q) > 0 {
				topQuantity = int(q)
				quantityExplicit = true
			}
		case int:
			if q > 0 {
				topQuantity = q
				quantityExplicit = true
			}
		case string:
			if parsed, err := strconv.Atoi(strings.TrimSpace(q)); err == nil && parsed > 0 {
				topQuantity = parsed
				quantityExplicit = true
			}
		}
	} else if qVal, ok := req.GetArguments()["qty"]; ok && qVal != nil {
		if q, ok := qVal.(float64); ok && int(q) > 0 {
			topQuantity = int(q)
			quantityExplicit = true
		}
	} else if qVal, ok := req.GetArguments()["count"]; ok && qVal != nil {
		if q, ok := qVal.(float64); ok && int(q) > 0 {
			topQuantity = int(q)
			quantityExplicit = true
		}
	} else if qVal, ok := req.GetArguments()["jumlah"]; ok && qVal != nil {
		if q, ok := qVal.(float64); ok && int(q) > 0 {
			topQuantity = int(q)
			quantityExplicit = true
		}
	}
	if topQuantity <= 0 {
		topQuantity = 1
	}

	type ReqItem struct {
		AssetID      string `json:"asset_id"`
		AssetName    string `json:"asset_name,omitempty"`
		DurationDays int    `json:"duration_days"`
	}

	var items []ReqItem

	// 1. Parse items argument (array of objects or array of strings)
	if rawItems, ok := req.GetArguments()["items"]; ok && rawItems != nil {
		switch v := rawItems.(type) {
		case []any:
			for _, elem := range v {
				switch e := elem.(type) {
				case string:
					if s := strings.TrimSpace(e); s != "" {
						items = append(items, ReqItem{AssetID: s, DurationDays: durationDays})
					}
				case map[string]any:
					aID := ""
					for _, k := range []string{"asset_id", "id", "asset", "code"} {
						if val, ok := e[k].(string); ok && strings.TrimSpace(val) != "" {
							aID = strings.TrimSpace(val)
							break
						}
					}
					aName := ""
					for _, k := range []string{"asset_name", "name", "nama", "title"} {
						if val, ok := e[k].(string); ok && strings.TrimSpace(val) != "" {
							aName = strings.TrimSpace(val)
							break
						}
					}
					dur := durationDays
					for _, k := range []string{"duration_days", "duration", "durasi", "days"} {
						if d, ok := e[k].(float64); ok && int(d) > 0 {
							dur = int(d)
							break
						}
					}
					itemQty := 1
					for _, k := range []string{"quantity", "qty", "count", "jumlah"} {
						if q, ok := e[k].(float64); ok && int(q) > 0 {
							itemQty = int(q)
							break
						}
					}
					if aID == "" && aName != "" {
						aID = aName
					}
				if aID != "" {
					if itemQty > 1 {
						return client.ErrResult("strict quantity: requested " + strconv.Itoa(itemQty) + "x '" + aID + "' but one asset_id is a single physical asset; call get_assets first and pass distinct IDs in items")
					}
					items = append(items, ReqItem{
						AssetID:      aID,
						AssetName:    aName,
						DurationDays: dur,
					})
				}
				}
			}
		default:
			b, err := json.Marshal(rawItems)
			if err == nil {
				_ = json.Unmarshal(b, &items)
			}
		}
	}

	// 2. Parse asset_ids / assets array arguments if items is still empty
	for _, argKey := range []string{"asset_ids", "assets"} {
		if rawList, ok := req.GetArguments()[argKey]; ok && rawList != nil {
			if list, ok := rawList.([]any); ok {
				for _, elem := range list {
					if s, ok := elem.(string); ok && strings.TrimSpace(s) != "" {
						items = append(items, ReqItem{
							AssetID:      strings.TrimSpace(s),
							DurationDays: durationDays,
						})
					}
				}
			}
		}
	}

	// 3. Parse single or comma/semicolon-separated asset_id
	// STRICT: never duplicate the same asset_id to fill quantity. One physical asset = one ID.
	if assetID != "" {
		parts := strings.FieldsFunc(assetID, func(r rune) bool {
			return r == ',' || r == ';'
		})
		if len(parts) > 1 {
			for _, part := range parts {
				if s := strings.TrimSpace(part); s != "" {
					items = append(items, ReqItem{
						AssetID:      s,
						DurationDays: durationDays,
					})
				}
			}
		} else if len(items) == 0 {
			if topQuantity > 1 {
				return client.ErrResult("strict quantity: requested " + strconv.Itoa(topQuantity) + " assets but only one distinct asset_id '" + strings.TrimSpace(assetID) + "' was supplied; call get_assets first and pass distinct IDs in items")
			}
			items = append(items, ReqItem{
				AssetID:      strings.TrimSpace(assetID),
				DurationDays: durationDays,
			})
		}
	}

	if len(items) == 0 {
		return client.ErrResult("asset_id or items is required to create a borrow transaction")
	}

	// STRICT: deduplicate by asset_id (case-insensitive); duplicates do not count toward quantity.
	seen := make(map[string]bool, len(items))
	distinct := make([]ReqItem, 0, len(items))
	for _, it := range items {
		key := strings.ToUpper(strings.TrimSpace(it.AssetID))
		if key == "" {
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		it.AssetID = strings.TrimSpace(it.AssetID)
		distinct = append(distinct, it)
	}
	items = distinct

	if len(items) == 0 {
		return client.ErrResult("asset_id or items is required to create a borrow transaction")
	}

	// STRICT: user-specified quantity must match distinct IDs supplied.
	if quantityExplicit && len(items) < topQuantity {
		return client.ErrResult("strict quantity: requested " + strconv.Itoa(topQuantity) + " assets but only " + strconv.Itoa(len(items)) + " distinct asset IDs were supplied; call get_assets first and pass distinct IDs in items")
	}

	// Validate items have valid duration
	for i := range items {
		if items[i].DurationDays <= 0 {
			items[i].DurationDays = durationDays
		}
	}

	// borrower_name is set from AI input; fallback to credentials name if not provided
	if borrowerName == "" || borrowerName == "-" || strings.EqualFold(borrowerName, "user") {
		if creds != nil && creds.NormalizedName() != "" {
			borrowerName = creds.NormalizedName()
		} else if creds != nil && creds.NormalizedUserID() != "" {
			borrowerName = creds.NormalizedUserID()
		} else {
			borrowerName = "user"
		}
	}

	// User Requirement: "ketika mcp tools ini dipanggil maka cari dulu id dari borrower_name masukan ke borrower_id"
	// Always look up users.id from DB based on borrower_name and assign to borrower_id
	lookupParams := url.Values{}
	lookupParams.Set("name", borrowerName)
	lookupResp, errLookup := client.DoRequest("GET", "/users/lookup", lookupParams, nil, creds)
	if errLookup == nil && lookupResp != nil {
		if m, ok := lookupResp.(map[string]any); ok {
			if uid, ok := m["user_id"].(string); ok && strings.TrimSpace(uid) != "" {
				borrowerID = strings.TrimSpace(uid)
			}
			if uname, ok := m["username"].(string); ok && strings.TrimSpace(uname) != "" {
				borrowerName = strings.TrimSpace(uname)
			}
			if comp, ok := m["company"].(string); ok && strings.TrimSpace(comp) != "" && groupName == "" {
				groupName = strings.TrimSpace(comp)
			}
		}
	}
	if client.Logger != nil {
		client.Logger.Printf("[HandleCreateBorrowTransaction] Resolved borrower_id=%s from DB for borrower_name=%s", borrowerID, borrowerName)
	}

	if groupName == "" {
		if creds != nil && creds.NormalizedCompany() != "" {
			groupName = creds.NormalizedCompany()
		} else {
			groupName = "qtera mandiri"
		}
	}

	if client.Logger != nil {
		client.Logger.Printf("[HandleCreateBorrowTransaction] Creating borrow request | borrower_id=%s | manager_id=%s | borrower_name=%s | manager_name=%s | group=%s | items=%d", borrowerID, managerID, borrowerName, managerName, groupName, len(items))
	}

	payload := map[string]any{
		"borrower_id":   borrowerID,
		"borrower_name": borrowerName,
		"manager_id":    managerID,
		"manager_name":  managerName,
		"group_name":    groupName,
		"items":         items,
	}

	resp, err := client.DoRequest("POST", "/borrow/hold", nil, payload, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(resp)
}

// HandleGetBorrowTransactions retrieves borrow transactions or a specific transaction by ID/code.
func HandleGetBorrowTransactions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	id := client.GetArgString(req, "id", "transaction_id", "transaction_code", "code")

	var path string
	params := url.Values{}

	if creds != nil {
		if creds.NormalizedCompany() != "" {
			params.Set("group_name", creds.NormalizedCompany())
		}
		if creds.NormalizedRole() != "admin" {
			if creds.NormalizedUserID() != "" {
				params.Set("borrower_id", creds.NormalizedUserID())
			}
		}
		params.Set("role", creds.NormalizedRole())
	}

	if id != "" {
		path = "/borrow/" + url.PathEscape(id)
	} else {
		path = "/borrow/approvals"
	}

	data, err := client.DoRequest("GET", path, params, nil, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(data)
}

// HandleGetBorrowHistoryRecommendations retrieves asset recommendations based on past borrow transactions.
func HandleGetBorrowHistoryRecommendations(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	query := client.GetArgString(req, "query", "keyword", "q", "asset", "name")
	category := client.GetArgString(req, "category")
	location := client.GetArgString(req, "location")
	groupName := client.GetArgString(req, "group_name")
	limit := 5
	if rawLimit, ok := req.GetArguments()["limit"]; ok && rawLimit != nil {
		switch v := rawLimit.(type) {
		case float64:
			if int(v) > 0 {
				limit = int(v)
			}
		case int:
			if v > 0 {
				limit = v
			}
		case string:
			if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && parsed > 0 {
				limit = parsed
			}
		}
	}

	params := url.Values{}
	if query != "" {
		params.Set("query", query)
	}
	if category != "" {
		params.Set("category", category)
	}
	if location != "" {
		params.Set("location", location)
	}
	if groupName != "" {
		params.Set("group_name", groupName)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	data, err := client.DoRequest("GET", "/borrow/history/recommendations", params, nil, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(data)
}

// HandleCheckBorrowAnomalies inspects anomaly signals and recommendations for a borrow transaction.
func HandleCheckBorrowAnomalies(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	id := client.GetArgString(req, "id", "transaction_id", "transaction_code", "code")
	if id == "" {
		return client.ErrResult("id (transaction ID or code) is required to inspect anomalies")
	}

	path := "/borrow/" + url.PathEscape(id) + "/anomalies"
	params := url.Values{}
	if creds != nil {
		if creds.NormalizedCompany() != "" {
			params.Set("group_name", creds.NormalizedCompany())
		}
		if creds.NormalizedRole() != "admin" {
			if creds.NormalizedUserID() != "" {
				params.Set("borrower_id", creds.NormalizedUserID())
			}
		}
		params.Set("role", creds.NormalizedRole())
	}

	data, err := client.DoRequest("GET", path, params, nil, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(data)
}

// HandleGetBorrowThresholdSettings retrieves the active borrow thresholds for the tenant.
func HandleGetBorrowThresholdSettings(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	params := url.Values{}
	if creds != nil {
		if creds.NormalizedCompany() != "" {
			params.Set("group_name", creds.NormalizedCompany())
		}
		params.Set("role", creds.NormalizedRole())
	}
	if group := client.GetArgString(req, "group_name", "company", "perusahaan"); group != "" {
		params.Set("group_name", group)
	}

	data, err := client.DoRequest("GET", "/borrow/anomalies/settings", params, nil, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(data)
}

// HandleUpdateBorrowThresholdSettings configures borrow thresholds (ADMIN ONLY).
func HandleUpdateBorrowThresholdSettings(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	if creds == nil || creds.NormalizedRole() != "admin" {
		return client.ErrResult("Akses ditolak: Hanya pengguna dengan role Administrator yang diizinkan untuk mengubah konfigurasi dan threshold peminjaman.")
	}

	payload := make(map[string]any)

	// Helper to extract integer from arguments
	getInt := func(keys ...string) (int, bool) {
		for _, k := range keys {
			if val, ok := req.GetArguments()[k]; ok && val != nil {
				switch v := val.(type) {
				case float64:
					if int(v) > 0 {
						return int(v), true
					}
				case int:
					if v > 0 {
						return v, true
					}
				case string:
					if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && parsed > 0 {
						return parsed, true
					}
				}
			}
		}
		return 0, false
	}

	if v, ok := getInt("max_duration_days", "duration_days", "max_duration", "durasi"); ok {
		payload["max_duration_days"] = v
	}
	if v, ok := getInt("max_active_loans", "active_loans", "max_loans"); ok {
		payload["max_active_loans"] = v
	}
	if v, ok := getInt("max_late_returns", "late_returns"); ok {
		payload["max_late_returns"] = v
	}
	if v, ok := getInt("max_extensions", "extensions"); ok {
		payload["max_extensions"] = v
	}
	if v, ok := getInt("max_asset_count", "asset_count", "max_assets"); ok {
		payload["max_asset_count"] = v
	}
	if v, ok := getInt("audit_window_days", "audit_window"); ok {
		payload["audit_window_days"] = v
	}
	if v, ok := getInt("repeat_borrow_cycles", "repeat_cycles"); ok {
		payload["repeat_borrow_cycles"] = v
	}
	if v, ok := getInt("min_signals_for_review", "min_signals"); ok {
		payload["min_signals_for_review"] = v
	}

	groupName := client.GetArgString(req, "group_name", "company", "perusahaan")
	if groupName != "" {
		payload["company_name"] = groupName
	} else if creds.NormalizedCompany() != "" {
		payload["company_name"] = creds.NormalizedCompany()
	}

	if len(payload) == 0 {
		return client.ErrResult("Harap tentukan setidaknya satu parameter threshold yang ingin diubah (contoh: max_duration_days, max_active_loans, dll).")
	}

	resp, err := client.DoRequest("POST", "/borrow/anomalies/settings", nil, payload, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(resp)
}
