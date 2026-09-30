package controller

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp_server/client"
)

// RegisterAssetTools mounts all asset-related MCP tools onto the MCP server.
func RegisterAssetTools(s *server.MCPServer) {
	// 1. get_assets: retrieve all assets for company inventory
	s.AddTool(mcp.NewTool("get_assets",
		mcp.WithDescription(
			`Fetch and retrieve all IT assets and equipment from the company inventory.
Always call this tool directly without situational filter parameters whenever querying, searching, counting, or checking assets. All records are retrieved so that extractContext can filter and analyze them accurately.
Security: Accessible to all roles.`),
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
  - location: Physical location (e.g. 'Head Office 3rd Floor', 'Warehouse A')
  - condition: Physical condition of the asset: 'Normal', 'Damaged', or 'Missing' (default 'Normal')
  - status: Operational status: 'active', 'in repair', 'retired' (default 'active')`),
		mcp.WithString("name", mcp.Required(), mcp.Description("Asset name (required)")),
		mcp.WithString("category", mcp.Required(), mcp.Description("Asset category classification (required)")),
		mcp.WithString("brand", mcp.Description("Asset brand or manufacturer")),
		mcp.WithString("modelType", mcp.Description("Model or specification type")),
		mcp.WithString("purchaseDate", mcp.Description("Purchase date string in YYYY-MM-DD format")),
		mcp.WithString("purchasePrice", mcp.Description("Purchase price amount")),
		mcp.WithString("location", mcp.Description("Physical asset location")),
		mcp.WithString("condition", mcp.Description("Physical condition: 'Normal', 'Damaged', or 'Missing' (default 'Normal')")),
		mcp.WithString("status", mcp.Description("Operational status: 'active', 'in repair', 'retired' (default 'active')")),
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
  - status: Updated asset status (e.g. 'active', 'in repair', 'retired')
  - condition: Updated physical condition ('Normal', 'Damaged', 'Missing')`),
		mcp.WithString("id", mcp.Required(), mcp.Description("Target unique asset identifier (required, e.g. 'AST-XXXXXXXX')")),
		mcp.WithString("name", mcp.Description("Updated asset name")),
		mcp.WithString("category", mcp.Description("Updated asset category")),
		mcp.WithString("brand", mcp.Description("Updated brand")),
		mcp.WithString("modelType", mcp.Description("Updated model or type")),
		mcp.WithString("purchaseDate", mcp.Description("Updated purchase date in YYYY-MM-DD format")),
		mcp.WithString("purchasePrice", mcp.Description("Updated purchase price")),
		mcp.WithString("location", mcp.Description("Updated location")),
		mcp.WithString("status", mcp.Description("Updated status (e.g. 'active', 'in repair', 'retired')")),
		mcp.WithString("condition", mcp.Description("Updated physical condition ('Normal', 'Damaged', 'Missing')")),
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

// HandleGetAssets retrieves all company assets without situational filtering so extractContext can filter accurately.
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
		client.Logger.Printf("[HandleGetAssets] Executing get_assets | perusahaan=%s (fetching all assets for extractContext)", company)
	}

	params := url.Values{}
	params.Set("perusahaan", company)
	params.Set("pageSize", "all")

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
		category = "-"
	}
	if location == "" {
		location = "-"
	}
	if brand == "" {
		brand = "-"
	}
	if modelType == "" {
		modelType = "-"
	}
	if purchaseDate == "" {
		purchaseDate = "-"
	}

	args := req.GetArguments()
	var purchasePriceStr string = "-"
	if v, ok := args["purchasePrice"]; ok && v != nil {
		purchasePriceStr = fmt.Sprintf("%v", v)
	} else if v, ok := args["purchase_price"]; ok && v != nil {
		purchasePriceStr = fmt.Sprintf("%v", v)
	} else if v, ok := args["price"]; ok && v != nil {
		purchasePriceStr = fmt.Sprintf("%v", v)
	}
	if purchasePriceStr == "" || purchasePriceStr == "<nil>" {
		purchasePriceStr = "-"
	}

	condition := client.GetArgString(req, "condition", "kondisi", "keadaan")
	switch strings.ToLower(condition) {
	case "damaged", "rusak", "damage":
		condition = "Damaged"
	case "missing", "hilang":
		condition = "Missing"
	case "normal", "baik", "good":
		condition = "Normal"
	default:
		if condition == "" || condition == "-" {
			condition = "Normal"
		}
	}

	status := client.GetArgString(req, "status")
	if status == "" || status == "-" {
		status = "active"
	}

	if client.Logger != nil {
		client.Logger.Printf("[HandleAddAsset] Creating asset | perusahaan=%s | name=%s | condition=%s | status=%s", company, name, condition, status)
	}

	body := map[string]any{
		"name":          name,
		"category":      category,
		"brand":         brand,
		"modelType":     modelType,
		"purchaseDate":  purchaseDate,
		"purchasePrice": purchasePriceStr,
		"location":      location,
		"perusahaan":    company,
		"status":        status,
		"condition":     condition,
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
	if condition := client.GetArgString(req, "condition", "kondisi", "keadaan"); condition != "" {
		switch strings.ToLower(condition) {
		case "damaged", "rusak", "damage":
			condition = "Damaged"
		case "missing", "hilang":
			condition = "Missing"
		case "normal", "baik", "good":
			condition = "Normal"
		}
		body["condition"] = condition
	}

	args := req.GetArguments()
	if v, ok := args["purchasePrice"]; ok && v != nil {
		body["purchasePrice"] = fmt.Sprintf("%v", v)
	} else if v, ok := args["purchase_price"]; ok && v != nil {
		body["purchasePrice"] = fmt.Sprintf("%v", v)
	} else if v, ok := args["price"]; ok && v != nil {
		body["purchasePrice"] = fmt.Sprintf("%v", v)
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
