package controller

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp_server/client"
)

// RegisterAssetTools mounts all asset-related MCP tools onto the MCP server.
func RegisterAssetTools(s *server.MCPServer) {
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
	), HandleGetAssets)

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
	), HandleAddAsset)

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
	), HandleUpdateAsset)

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
	), HandleDeleteAsset)
}

// HandleGetAssets queries the assets inventory with optional search, filters, sorting, and pagination.
func HandleGetAssets(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	company := creds.NormalizedCompany()

	if company == "" {
		if client.Logger != nil {
			client.Logger.Println("[HandleGetAssets] Error: company credentials missing from request context")
		}
		return client.ErrResult("tenant company context is missing from request credentials")
	}

	if client.Logger != nil {
		client.Logger.Printf("[HandleGetAssets] Executing get_assets | perusahaan=%s", company)
	}

	params := url.Values{}
	params.Set("perusahaan", company)

	if search := client.GetArgString(req, "search", "query", "q"); search != "" {
		params.Set("search", search)
	}
	if category := client.GetArgString(req, "category", "categories"); category != "" {
		params.Set("category", category)
	}
	if brand := client.GetArgString(req, "brand", "brands"); brand != "" {
		params.Set("brand", brand)
	}
	if location := client.GetArgString(req, "location", "locations"); location != "" {
		params.Set("location", location)
	}
	if sort := client.GetArgString(req, "sort", "sort_by", "sortBy"); sort != "" {
		params.Set("sort", sort)
	}
	if order := client.GetArgString(req, "order", "direction"); order != "" {
		params.Set("order", strings.ToUpper(order))
	}
	if pageSize := client.GetArgString(req, "pageSize", "page_size", "limit"); pageSize != "" {
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

	data, err := client.DoRequest("GET", "/assets", params, nil, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(data)
}

// HandleAddAsset registers a new asset. Automatically bound to user's perusahaan.
func HandleAddAsset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	company := creds.NormalizedCompany()

	if company == "" {
		if client.Logger != nil {
			client.Logger.Println("[HandleAddAsset] Error: company credentials missing from request context")
		}
		return client.ErrResult("tenant company context is missing from request credentials")
	}

	// RBAC: Operator is restricted to read-only operations
	if creds != nil && creds.NormalizedRole() == "operator" {
		return client.ErrResult("forbidden: operator role is not authorized to register assets")
	}

	name := client.GetArgString(req, "name", "asset_name")
	category := client.GetArgString(req, "category")
	brand := client.GetArgString(req, "brand")
	modelType := client.GetArgString(req, "modelType", "model_type", "type")
	purchaseDate := client.GetArgString(req, "purchaseDate", "purchase_date", "date")
	location := client.GetArgString(req, "location")

	if name == "" {
		return client.ErrResult("name is required and cannot be empty")
	}
	if category == "" {
		return client.ErrResult("category is required and cannot be empty")
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

	if client.Logger != nil {
		client.Logger.Printf("[HandleAddAsset] Creating asset | perusahaan=%s | name=%s", company, name)
	}

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

	data, err := client.DoRequest("POST", "/assets", nil, body, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(data)
}

// HandleUpdateAsset updates fields of an existing asset identified by id.
func HandleUpdateAsset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	company := creds.NormalizedCompany()

	if company == "" {
		if client.Logger != nil {
			client.Logger.Println("[HandleUpdateAsset] Error: company credentials missing from request context")
		}
		return client.ErrResult("tenant company context is missing from request credentials")
	}

	// RBAC: Operator is restricted to read-only operations
	if creds != nil && creds.NormalizedRole() == "operator" {
		return client.ErrResult("forbidden: operator role is not authorized to modify assets")
	}

	id := client.GetArgString(req, "id", "asset_id", "assetId")
	if id == "" {
		return client.ErrResult("'id' is required")
	}

	body := make(map[string]any)
	if name := client.GetArgString(req, "name", "asset_name"); name != "" {
		body["name"] = name
	}
	if category := client.GetArgString(req, "category"); category != "" {
		body["category"] = category
	}
	if brand := client.GetArgString(req, "brand"); brand != "" {
		body["brand"] = brand
	}
	if modelType := client.GetArgString(req, "modelType", "model_type", "type"); modelType != "" {
		body["modelType"] = modelType
	}
	if purchaseDate := client.GetArgString(req, "purchaseDate", "purchase_date", "date"); purchaseDate != "" {
		body["purchaseDate"] = purchaseDate
	}
	if location := client.GetArgString(req, "location"); location != "" {
		body["location"] = location
	}
	if status := client.GetArgString(req, "status"); status != "" {
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

	data, err := client.DoRequest("PATCH", "/assets/"+url.PathEscape(id), nil, body, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(data)
}

// HandleDeleteAsset deletes an asset by its id.
func HandleDeleteAsset(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	company := creds.NormalizedCompany()

	if company == "" {
		if client.Logger != nil {
			client.Logger.Println("[HandleDeleteAsset] Error: company credentials missing from request context")
		}
		return client.ErrResult("tenant company context is missing from request credentials")
	}

	// RBAC: Operator is restricted to read-only operations
	if creds != nil && creds.NormalizedRole() == "operator" {
		return client.ErrResult("forbidden: operator role is not authorized to delete assets")
	}

	id := client.GetArgString(req, "id", "asset_id", "assetId")
	if id == "" {
		return client.ErrResult("'id' is required")
	}

	data, err := client.DoRequest("DELETE", "/assets/"+url.PathEscape(id), nil, nil, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(data)
}
