package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ---------- config / logging ----------

var (
	baseURL = getEnv("BASE_URL", "http://127.0.0.1:3000/api")
	apiKey  = getEnv("API_KEY", "")
	logger  *log.Logger
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func initLogger() {
	f, err := os.OpenFile("mcp_server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	logger = log.New(f, "", log.LstdFlags|log.Lshortfile)
	logger.Println("mcp server started")
	if apiKey == "" {
		logger.Println("WARNING: API_KEY is empty — requests to BASE_URL will likely fail auth if protected")
	}
}

// ---------- credentials (injected by server, not by the LLM) ----------

// Credentials mirrors the user context injected by the server/session layer.
type Credentials struct {
	Token   string `json:"token,omitempty"`
	UserID  string `json:"user_id,omitempty"`
	Company string `json:"company,omitempty"`
	Role    string `json:"role,omitempty"`
}

func (c *Credentials) NormalizedRole() string {
	if c == nil || strings.TrimSpace(c.Role) == "" {
		return "operator"
	}
	return strings.ToLower(strings.TrimSpace(c.Role))
}

func (c *Credentials) NormalizedCompany() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.Company)
}

func buildHeaders(creds *Credentials) http.Header {
	h := http.Header{}
	if apiKey != "" {
		h.Set("Authorization", "Bearer "+apiKey)
	}
	if creds != nil {
		if creds.Token != "" {
			h.Set("Authorization", "Bearer "+creds.Token)
		}
		if creds.UserID != "" {
			h.Set("X-User-ID", creds.UserID)
		}
		if creds.Company != "" {
			h.Set("X-Company", creds.Company)
		}
		if creds.Role != "" {
			h.Set("X-Role", creds.Role)
		}
	}
	return h
}

// ---------- HTTP client ----------

var httpClient = &http.Client{Timeout: 10 * time.Second}

func doRequest(method, path string, params url.Values, body any, creds *Credentials) (any, error) {
	u := strings.TrimRight(baseURL, "/") + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = strings.NewReader(string(b))
	}

	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header = buildHeaders(creds)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Printf("%s %s failed: %v", method, u, err)
		return map[string]string{"error": err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("%s %s returned status %d", method, u, resp.StatusCode)
		logger.Println(errMsg)
		return map[string]string{"error": errMsg}, nil
	}

	if resp.StatusCode == http.StatusNoContent {
		return map[string]any{"ok": true, "status": "deleted"}, nil
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Printf("%s %s read failed: %v", method, u, err)
		return map[string]string{"error": err.Error()}, nil
	}

	if len(respBytes) == 0 {
		return map[string]any{"ok": true}, nil
	}

	var parsed any
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		// Not JSON — return raw string rather than failing the tool call.
		return string(respBytes), nil
	}
	return parsed, nil
}

// ---------- helpers ----------

func extractCredentials(req mcp.CallToolRequest) *Credentials {
	raw, ok := req.GetArguments()["credentials"]
	if !ok || raw == nil {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var creds Credentials
	if err := json.Unmarshal(b, &creds); err != nil {
		return nil
	}
	return &creds
}

func getArgString(req mcp.CallToolRequest, keys ...string) string {
	args := req.GetArguments()
	if args == nil {
		return ""
	}
	for _, key := range keys {
		if val, ok := args[key]; ok && val != nil {
			if s, ok := val.(string); ok {
				trimmed := strings.TrimSpace(s)
				if trimmed != "" {
					return trimmed
				}
			} else {
				s := strings.TrimSpace(fmt.Sprintf("%v", val))
				if s != "" && s != "<nil>" {
					return s
				}
			}
		}
	}
	return ""
}

func toolResult(data any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func errResult(msg string) (*mcp.CallToolResult, error) {
	return toolResult(map[string]string{"error": msg})
}

// ---------- asset tool handlers with role & perusahaan security --------// handleGetAssets queries the assets inventory with optional search, filters, sorting, and pagination.
// Security: Accessible to admin, operator, and viewer roles. Strictly scoped to user's perusahaan.
func handleGetAssets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := extractCredentials(req)
	company := creds.NormalizedCompany()

	if company == "" {
		logger.Println("[handleGetAssets] Error: company credentials missing from request context")
		return errResult("tenant company context is missing from request credentials")
	}

	logger.Printf("[handleGetAssets] Executing get_assets | perusahaan=%s", company)

	params := url.Values{}
	params.Set("perusahaan", company)

	if search := getArgString(req, "search", "query", "q"); search != "" {
		params.Set("search", search)
	}
	if category := getArgString(req, "category", "categories"); category != "" {
		params.Set("category", category)
	}
	if brand := getArgString(req, "brand", "brands"); brand != "" {
		params.Set("brand", brand)
	}
	if location := getArgString(req, "location", "locations"); location != "" {
		params.Set("location", location)
	}
	if sort := getArgString(req, "sort", "sort_by", "sortBy"); sort != "" {
		params.Set("sort", sort)
	}
	if order := getArgString(req, "order", "direction"); order != "" {
		params.Set("order", strings.ToUpper(order))
	}
	if pageSize := getArgString(req, "pageSize", "page_size", "limit"); pageSize != "" {
		params.Set("pageSize", pageSize)
	}

	args := req.GetArguments()
	if pageVal, ok := args["page"]; ok && pageVal != nil {
		switch v := pageVal.(type) {
		case float64:
			params.Set("page", strconv.Itoa(int(v)))
		case int:
			params.Set("page", strconv.Itoa(v))
		case string:
			if strings.TrimSpace(v) != "" {
				params.Set("page", strings.TrimSpace(v))
			}
		}
	}

	data, err := doRequest("GET", "/assets", params, nil, creds)
	if err != nil {
		return errResult(err.Error())
	}
	return toolResult(data)
}

// handleAddAsset registers a new asset. Automatically bound to user's perusahaan.
func handleAddAsset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := extractCredentials(req)
	company := creds.NormalizedCompany()

	if company == "" {
		logger.Println("[handleAddAsset] Error: company credentials missing from request context")
		return errResult("tenant company context is missing from request credentials")
	}

	name := getArgString(req, "name", "asset_name")
	category := getArgString(req, "category")
	brand := getArgString(req, "brand")
	modelType := getArgString(req, "modelType", "model_type", "type")
	purchaseDate := getArgString(req, "purchaseDate", "purchase_date", "date")
	location := getArgString(req, "location")

	if name == "" {
		return errResult("name is required and cannot be empty")
	}
	if category == "" {
		return errResult("category is required and cannot be empty")
	}

	args := req.GetArguments()
	var purchasePrice any
	if v, ok := args["purchasePrice"]; ok && v != nil {
		purchasePrice = v
	} else if v, ok := args["purchase_price"]; ok && v != nil {
		purchasePrice = v
	} else if v, ok := args["price"]; ok && v != nil {
		purchasePrice = v
	}

	logger.Printf("[handleAddAsset] Creating asset | perusahaan=%s | name=%s", company, name)

	body := map[string]any{
		"name":          name,
		"category":      category,
		"brand":         brand,
		"modelType":     modelType,
		"purchaseDate":  purchaseDate,
		"purchasePrice": purchasePrice,
		"location":      location,
		"perusahaan":    company,
		"status":        "active",
	}

	data, err := doRequest("POST", "/assets", nil, body, creds)
	if err != nil {
		return errResult(err.Error())
	}
	return toolResult(data)
}

// handleUpdateAsset updates fields of an existing asset identified by id.
func handleUpdateAsset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := extractCredentials(req)
	company := creds.NormalizedCompany()

	if company == "" {
		logger.Println("[handleUpdateAsset] Error: company credentials missing from request context")
		return errResult("tenant company context is missing from request credentials")
	}

	id := getArgString(req, "id", "asset_id", "assetId")
	if id == "" {
		return errResult("'id' is required")
	}

	body := make(map[string]any)
	if name := getArgString(req, "name", "asset_name"); name != "" {
		body["name"] = name
	}
	if category := getArgString(req, "category"); category != "" {
		body["category"] = category
	}
	if brand := getArgString(req, "brand"); brand != "" {
		body["brand"] = brand
	}
	if modelType := getArgString(req, "modelType", "model_type", "type"); modelType != "" {
		body["modelType"] = modelType
	}
	if purchaseDate := getArgString(req, "purchaseDate", "purchase_date", "date"); purchaseDate != "" {
		body["purchaseDate"] = purchaseDate
	}
	if location := getArgString(req, "location"); location != "" {
		body["location"] = location
	}
	if status := getArgString(req, "status"); status != "" {
		body["status"] = status
	}

	args := req.GetArguments()
	if v, ok := args["purchasePrice"]; ok && v != nil {
		body["purchasePrice"] = v
	} else if v, ok := args["purchase_price"]; ok && v != nil {
		body["purchasePrice"] = v
	} else if v, ok := args["price"]; ok && v != nil {
		body["purchasePrice"] = v
	}

	data, err := doRequest("PATCH", "/assets/"+url.PathEscape(id), nil, body, creds)
	if err != nil {
		return errResult(err.Error())
	}
	return toolResult(data)
}

// handleDeleteAsset deletes an asset by its id.
func handleDeleteAsset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := extractCredentials(req)
	company := creds.NormalizedCompany()

	if company == "" {
		logger.Println("[handleDeleteAsset] Error: company credentials missing from request context")
		return errResult("tenant company context is missing from request credentials")
	}

	id := getArgString(req, "id", "asset_id", "assetId")
	if id == "" {
		return errResult("'id' is required")
	}

	data, err := doRequest("DELETE", "/assets/"+url.PathEscape(id), nil, nil, creds)
	if err != nil {
		return errResult(err.Error())
	}
	return toolResult(data)
}

// parseTableArgs extracts headers, data rows, title, filename, and sheet arguments safely from CallToolRequest
func parseTableArgs(req mcp.CallToolRequest) (headers []string, data [][]string, title string, filename string, sheet string) {
	args := req.GetArguments()
	if args == nil {
		return nil, nil, "", "", ""
	}

	title = getArgString(req, "title", "name")
	filename = getArgString(req, "filename", "file_name")
	sheet = getArgString(req, "sheet", "sheet_name")

	// Parse headers
	if rawHeaders, ok := args["headers"]; ok && rawHeaders != nil {
		switch v := rawHeaders.(type) {
		case []any:
			for _, item := range v {
				headers = append(headers, fmt.Sprintf("%v", item))
			}
		case []string:
			headers = v
		case string:
			if err := json.Unmarshal([]byte(v), &headers); err != nil {
				for _, part := range strings.Split(v, ",") {
					if t := strings.TrimSpace(part); t != "" {
						headers = append(headers, t)
					}
				}
			}
		}
	}

	// Parse data rows
	if rawData, ok := args["data"]; ok && rawData != nil {
		var rawRows []any
		switch v := rawData.(type) {
		case []any:
			rawRows = v
		case []string:
			data = append(data, v)
		case string:
			var listAny []any
			if err := json.Unmarshal([]byte(v), &listAny); err == nil {
				rawRows = listAny
			} else {
				_ = json.Unmarshal([]byte(v), &data)
			}
		}

		// If headers were omitted but data is a list of maps, deduce headers from first row keys
		if len(headers) == 0 && len(rawRows) > 0 {
			if firstMap, ok := rawRows[0].(map[string]any); ok {
				for k := range firstMap {
					headers = append(headers, k)
				}
			}
		}

		for _, rowAny := range rawRows {
			var row []string
			switch r := rowAny.(type) {
			case []any:
				for _, cell := range r {
					row = append(row, fmt.Sprintf("%v", cell))
				}
			case []string:
				row = r
			case map[string]any:
				// Map dictionary keys to ordered headers
				for _, h := range headers {
					val := ""
					hClean := strings.TrimSpace(h)
					if v, exists := r[hClean]; exists {
						val = fmt.Sprintf("%v", v)
					} else {
						// Case-insensitive & normalized fallback search
						for k, kv := range r {
							kClean := strings.TrimSpace(k)
							if strings.EqualFold(kClean, hClean) {
								val = fmt.Sprintf("%v", kv)
								break
							}
						}
					}
					row = append(row, val)
				}
			}
			if len(row) > 0 {
				data = append(data, row)
			}
		}
	}

	return headers, data, title, filename, sheet
}

// handleGeneratePDF generates a formatted PDF table with watermark and returns its download URL.
func handleGeneratePDF(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := extractCredentials(req)
	headers, data, title, _, _ := parseTableArgs(req)

	if len(headers) == 0 || len(data) == 0 {
		return errResult("both 'headers' and 'data' are required to generate a PDF table")
	}

	logger.Printf("[handleGeneratePDF] Generating PDF | title=%s | cols=%d | rows=%d", title, len(headers), len(data))

	body := map[string]any{
		"title":   title,
		"headers": headers,
		"data":    data,
	}

	resp, err := doRequest("POST", "/pdf", nil, body, creds)
	if err != nil {
		return errResult(err.Error())
	}
	return toolResult(resp)
}

// handleGenerateCSV generates a CSV document from data rows and returns its download URL.
func handleGenerateCSV(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := extractCredentials(req)
	headers, data, title, filename, _ := parseTableArgs(req)

	if len(headers) == 0 || len(data) == 0 {
		return errResult("both 'headers' and 'data' are required to generate a CSV export")
	}

	logger.Printf("[handleGenerateCSV] Generating CSV | filename=%s | title=%s | cols=%d | rows=%d", filename, title, len(headers), len(data))

	body := map[string]any{
		"title":    title,
		"filename": filename,
		"headers":  headers,
		"data":     data,
	}

	resp, err := doRequest("POST", "/csv", nil, body, creds)
	if err != nil {
		return errResult(err.Error())
	}
	return toolResult(resp)
}

// handleGenerateExcel generates a styled Excel (.xlsx) spreadsheet and returns its download URL.
func handleGenerateExcel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := extractCredentials(req)
	headers, data, title, filename, sheet := parseTableArgs(req)

	if len(headers) == 0 || len(data) == 0 {
		return errResult("both 'headers' and 'data' are required to generate an Excel spreadsheet")
	}

	logger.Printf("[handleGenerateExcel] Generating Excel | filename=%s | sheet=%s | title=%s | cols=%d | rows=%d", filename, sheet, title, len(headers), len(data))

	body := map[string]any{
		"title":    title,
		"filename": filename,
		"sheet":    sheet,
		"headers":  headers,
		"data":     data,
	}

	resp, err := doRequest("POST", "/excel", nil, body, creds)
	if err != nil {
		return errResult(err.Error())
	}
	return toolResult(resp)
}

// handleNoTools executes when no external tools or database operations are needed.
func handleNoTools(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	reason := getArgString(req, "reason", "message", "explanation")
	logger.Printf("[handleNoTools] Executing no_tools | reason=%s", reason)
	return toolResult(map[string]any{
		"ok":      true,
		"status":  "no_tools_needed",
		"message": "No external tools or database operations required for this query.",
		"reason":  reason,
	})
}

// handleCreateSchedule inserts a new schedule record into public.schedule.
func handleCreateSchedule(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := extractCredentials(req)

	category := getArgString(req, "category", "nama_jadwal", "title", "name")
	start := getArgString(req, "start", "start_date", "tanggal_mulai", "tanggal")
	frequency := getArgString(req, "frequency", "frekuensi", "periode")

	if category == "" {
		return errResult("category is required and cannot be empty")
	}
	if start == "" {
		start = time.Now().Format("2006-01-02")
	}
	if frequency == "" {
		frequency = "Once"
	}

	logger.Printf("[handleCreateSchedule] Inserting schedule | category=%s | start=%s | frequency=%s", category, start, frequency)

	body := map[string]any{
		"category":  category,
		"start":     start,
		"frequency": frequency,
	}

	resp, err := doRequest("POST", "/schedule", nil, body, creds)
	if err != nil {
		return errResult(err.Error())
	}
	return toolResult(resp)
}

// handleGetSchedules retrieves schedule records from public.schedule.
func handleGetSchedules(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := extractCredentials(req)
	params := url.Values{}
	if category := getArgString(req, "category", "search"); category != "" {
		params.Set("category", category)
	}

	data, err := doRequest("GET", "/schedule", params, nil, creds)
	if err != nil {
		return errResult(err.Error())
	}
	return toolResult(data)
}

// ---------- server setup & tool registration ----------

func main() {
	initLogger()

	s := server.NewMCPServer(
		"qtera-inventory-mcp",
		"2.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	// 1. get_assets: query asset inventory
	s.AddTool(mcp.NewTool("get_assets",
		mcp.WithDescription(
			`Search, query, filter, and inspect IT assets and hardware in the company inventory.
Use this tool whenever the user asks to find, check, count, list, or examine assets, equipment, laptops, monitors, serial numbers, locations, or purchase history.
Company scoping is handled automatically.
Parameters:
  - search: Free text keyword matching asset name, asset ID, brand, or category
  - category: Filter by category (e.g. 'IT Equipment > Laptop', 'Monitor', 'Furniture')
  - brand: Filter by manufacturer/brand (e.g. 'Dell', 'Apple', 'Lenovo', 'HP')
  - location: Filter by physical storage or office location (e.g. 'Warehouse A', 'Office 2nd Floor')
  - sort: Column to sort by ('name', 'category', 'brand', 'purchaseDate', 'createdAt')
  - order: Sort direction ('ASC' or 'DESC')
  - pageSize: Number of records to return (default: 10, or 'all' to retrieve complete list)`),
		mcp.WithString("search", mcp.Description("Free text search query matching name, asset_id, category, or brand")),
		mcp.WithString("category", mcp.Description("Filter by category name (comma-separated for multiple)")),
		mcp.WithString("brand", mcp.Description("Filter by brand name (comma-separated for multiple)")),
		mcp.WithString("location", mcp.Description("Filter by location (comma-separated for multiple)")),
		mcp.WithString("sort", mcp.Description("Sort column: 'name', 'category', 'brand', 'purchaseDate', 'createdAt'")),
		mcp.WithString("order", mcp.Description("Sort direction: 'ASC' or 'DESC'")),
		mcp.WithString("pageSize", mcp.Description("Number of records per page: default '10', or 'all' for complete list")),
	), handleGetAssets)

	// 2. add_asset: create new asset
	s.AddTool(mcp.NewTool("add_asset",
		mcp.WithDescription(
			`Register, insert, or create a new asset record in the company inventory.
Use this tool whenever the user asks to add, create, insert, or import a new asset (or multiple assets from an attached document, spreadsheet, or invoice).
Security: Accessible to 'admin' and 'operator' roles.
Parameters:
  - name: Asset title/name (REQUIRED, e.g. 'Dell Latitude 5450', 'MacBook Pro 16')
  - category: Category classification (REQUIRED, e.g. 'IT Equipment > Laptop', 'Monitor', 'Furniture')
  - brand: Brand or manufacturer (e.g. 'Dell', 'Apple', 'HP')
  - modelType: Specific model or specification (e.g. 'Latitude 5450', 'M3 Max 36GB')
  - purchaseDate: Date of purchase in YYYY-MM-DD format (e.g. '2026-08-20')
  - purchasePrice: Purchase price amount as number or string (e.g. 15000000)
  - location: Physical location (e.g. 'Head Office 3rd Floor', 'Warehouse A')`),
		mcp.WithString("name", mcp.Required(), mcp.Description("Asset name (required)")),
		mcp.WithString("category", mcp.Required(), mcp.Description("Asset category classification (required)")),
		mcp.WithString("brand", mcp.Description("Asset brand or manufacturer")),
		mcp.WithString("modelType", mcp.Description("Model or specification type")),
		mcp.WithString("purchaseDate", mcp.Description("Purchase date string in YYYY-MM-DD format")),
		mcp.WithString("purchasePrice", mcp.Description("Purchase price amount")),
		mcp.WithString("location", mcp.Description("Physical asset location")),
	), handleAddAsset)

	// 3. update_asset: update asset by id
	s.AddTool(mcp.NewTool("update_asset",
		mcp.WithDescription(
			`Update or modify specific fields of an existing asset record by its unique asset ID.
Use this tool whenever the user asks to edit, update, relocate, reprice, or reclassify an asset.
Do NOT guess or alter the asset ID format (keep e.g. 'AST-6D2933A2' exactly as found).
Security: Accessible to 'admin' and 'operator' roles.
Parameters:
  - id: The exact unique asset identifier to update (REQUIRED, e.g. 'AST-XXXXXXXX')
  - name: Updated asset name
  - category: Updated category classification
  - brand: Updated brand or manufacturer
  - modelType: Updated model or type
  - purchaseDate: Updated purchase date in YYYY-MM-DD format
  - purchasePrice: Updated purchase price amount
  - location: Updated physical location
  - status: Updated asset status (e.g. 'Active', 'In Repair', 'Retired', 'Available')`),
		mcp.WithString("id", mcp.Required(), mcp.Description("Target unique asset identifier (required, e.g. 'AST-XXXXXXXX')")),
		mcp.WithString("name", mcp.Description("Updated asset name")),
		mcp.WithString("category", mcp.Description("Updated asset category")),
		mcp.WithString("brand", mcp.Description("Updated brand")),
		mcp.WithString("modelType", mcp.Description("Updated model or type")),
		mcp.WithString("purchaseDate", mcp.Description("Updated purchase date in YYYY-MM-DD format")),
		mcp.WithString("purchasePrice", mcp.Description("Updated purchase price")),
		mcp.WithString("location", mcp.Description("Updated location")),
	), handleUpdateAsset)

	// 4. delete_asset: delete asset by id
	s.AddTool(mcp.NewTool("delete_asset",
		mcp.WithDescription(
			`Permanently delete an asset record from the company inventory by its unique asset ID.
Use this tool ONLY when the user explicitly requests to delete, destroy, or remove an asset.
Do NOT guess the ID; ensure the exact asset ID (e.g. 'AST-6D2933A2') is provided.
Security: Accessible ONLY to 'admin' role. Operator and viewer requests are blocked.
Parameters:
  - id: The exact unique asset identifier to delete (REQUIRED, e.g. 'AST-XXXXXXXX')`),
		mcp.WithString("id", mcp.Required(), mcp.Description("Target unique asset identifier to delete (required, e.g. 'AST-XXXXXXXX')")),
	), handleDeleteAsset)

	// 5. generate_pdf: generate formatted PDF table report
	s.AddTool(mcp.NewTool("generate_pdf",
		mcp.WithDescription(
			`Generate an asset inventory report or professionally formatted PDF table document with custom headers, data rows, zebra row styling, and company watermark from company database queries.
Use this tool whenever the user asks to export, generate, create, download, or print an asset report, table, or inventory summary from company assets in PDF format.
Do NOT use this tool to convert uploaded user files, spreadsheets, or documents to PDF. File conversion is strictly unsupported.
Parameters:
  - headers: Array of column header titles (REQUIRED, e.g. ["No", "Nama Aset", "Kategori", "Lokasi", "Harga"])
  - data: 2D array of rows containing string values (REQUIRED, e.g. [["1", "ThinkPad T14", "Laptop", "Lantai 2", "Rp 18.000.000"]])
  - title: Optional document title for the PDF report`),
		mcp.WithArray("headers", mcp.Required(), mcp.Description("List of column header titles, e.g. ['No', 'Asset Name', 'Category', 'Location', 'Price']")),
		mcp.WithArray("data", mcp.Required(), mcp.Description("2D array of rows with table cell values")),
		mcp.WithString("title", mcp.Description("Optional title for the report")),
	), handleGeneratePDF)

	// 6. generate_csv: generate CSV document from database records
	s.AddTool(mcp.NewTool("generate_csv",
		mcp.WithDescription(
			`Generate an asset inventory report or structured table in CSV (Comma-Separated Values) format from company database records.
Use this tool whenever the user asks to export, generate, create, or download data in CSV format or CSV file from company assets.
Do NOT use this tool to convert uploaded user files, spreadsheets, or documents to CSV. File conversion is strictly unsupported.
Parameters:
  - headers: Array of column header titles (REQUIRED, e.g. ["No", "Nama Aset", "Kategori", "Lokasi", "Harga"])
  - data: 2D array of rows containing string values (REQUIRED, e.g. [["1", "ThinkPad T14", "Laptop", "Lantai 2", "Rp 18.000.000"]])
  - title: Optional document title for naming the CSV export
  - filename: Optional specific file name (e.g. "asset_report.csv")`),
		mcp.WithArray("headers", mcp.Required(), mcp.Description("List of column header titles, e.g. ['No', 'Asset Name', 'Category', 'Location', 'Price']")),
		mcp.WithArray("data", mcp.Required(), mcp.Description("2D array of rows with table cell values")),
		mcp.WithString("title", mcp.Description("Optional title for naming the report")),
		mcp.WithString("filename", mcp.Description("Optional filename for the CSV file")),
	), handleGenerateCSV)

	// 7. generate_excel: generate styled Microsoft Excel (.xlsx) report
	s.AddTool(mcp.NewTool("generate_excel",
		mcp.WithDescription(
			`Generate a professionally styled Microsoft Excel spreadsheet (.xlsx) with headers, formatted rows, and sheet names from company database records.
Use this tool whenever the user asks to export, generate, create, or download data in Excel (.xlsx) format or spreadsheet table from company assets.
Do NOT use this tool to convert uploaded user files, spreadsheets, or documents to Excel. File conversion is strictly unsupported.
Parameters:
  - headers: Array of column header titles (REQUIRED, e.g. ["No", "Nama Aset", "Kategori", "Lokasi", "Harga"])
  - data: 2D array of rows containing string values (REQUIRED, e.g. [["1", "ThinkPad T14", "Laptop", "Lantai 2", "Rp 18.000.000"]])
  - title: Optional report title displayed at the top of the spreadsheet
  - filename: Optional specific file name (e.g. "asset_inventory.xlsx")
  - sheet: Optional worksheet name (e.g. "Inventory")`),
		mcp.WithArray("headers", mcp.Required(), mcp.Description("List of column header titles, e.g. ['No', 'Asset Name', 'Category', 'Location', 'Price']")),
		mcp.WithArray("data", mcp.Required(), mcp.Description("2D array of rows with table cell values")),
		mcp.WithString("title", mcp.Description("Optional title displayed at the top")),
		mcp.WithString("filename", mcp.Description("Optional filename for the Excel file")),
		mcp.WithString("sheet", mcp.Description("Optional sheet name")),
	), handleGenerateExcel)

	// 8. create_schedule: insert record to public.schedule
	s.AddTool(mcp.NewTool("create_schedule",
		mcp.WithDescription(
			`Create and insert a new schedule record into the public.schedule database table.
Use this tool whenever the user asks to schedule, plan, or create an audit schedule, maintenance schedule, stock-take, or recurring task (e.g. 'buat jadwal audit laptop kantor mulai tanggal 15 September secara bulanan').
Parameters:
  - category: Schedule category or description (REQUIRED, max 125 chars, e.g. 'Audit Aset Laptop', 'Stock Opname Hardware', 'Audit Rutin')
  - start: Starting date or schedule timeframe (REQUIRED, max 125 chars, e.g. '2026-09-15', 'Senin Depan', '2026-10-01 09:00')
  - frequency: Recurrence frequency (REQUIRED, max 125 chars, e.g. 'Monthly', 'Weekly', 'Quarterly', 'Bulanan', 'Tahunan', 'Once')`),
		mcp.WithString("category", mcp.Required(), mcp.Description("Schedule category or description (required, max 125 chars)")),
		mcp.WithString("start", mcp.Required(), mcp.Description("Start date or start time string (required, max 125 chars, e.g. '2026-09-15')")),
		mcp.WithString("frequency", mcp.Required(), mcp.Description("Frequency or recurrence period (required, max 125 chars, e.g. 'Monthly', 'Weekly')")),
	), handleCreateSchedule)

	// 9. get_schedules: list schedules from public.schedule
	s.AddTool(mcp.NewTool("get_schedules",
		mcp.WithDescription(
			`List or view existing schedules from the public.schedule database table.
Use this tool whenever the user asks to check, view, or list existing audit schedules or tasks.
Parameters:
  - category: Optional category keyword filter`),
		mcp.WithString("category", mcp.Description("Optional category filter")),
	), handleGetSchedules)

	// 10. no_tools: tool for queries needing no external tools
	s.AddTool(mcp.NewTool("no_tools",
		mcp.WithDescription(
			`Call this tool when the task or question does NOT require any tools (e.g. greetings, casual conversation, general questions, or clarification that does not require querying or modifying the asset database).
This tool performs no action and signals that no external tools are needed for this turn.
Parameters:
  - reason: Optional explanation of why no tools are needed (e.g. 'General greeting', 'Casual conversation', 'Non-asset question')`),
		mcp.WithString("reason", mcp.Description("Optional explanation why no tools are needed")),
	), handleNoTools)

	if err := server.ServeStdio(s); err != nil {
		logger.Fatalf("server error: %v", err)
	}
}
