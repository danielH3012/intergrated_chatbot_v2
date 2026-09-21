package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp_server/client"
)

// RegisterDocumentTools mounts document and report generation tools onto the MCP server.
func RegisterDocumentTools(s *server.MCPServer) {
	// 1. generate_pdf: generate formatted PDF table report
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
	), HandleGeneratePDF)

	// 2. generate_csv: generate CSV document from database records
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
	), HandleGenerateCSV)

	// 3. generate_excel: generate styled Microsoft Excel (.xlsx) report
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
	), HandleGenerateExcel)
}

// ParseTableArgs extracts headers, data rows, title, filename, and sheet arguments safely from CallToolRequest
func ParseTableArgs(req mcp.CallToolRequest) (headers []string, data [][]string, title string, filename string, sheet string) {
	args := req.GetArguments()
	if args == nil {
		return nil, nil, "", "", ""
	}

	title = client.GetArgString(req, "title", "name")
	filename = client.GetArgString(req, "filename", "file_name")
	sheet = client.GetArgString(req, "sheet", "sheet_name")

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

// HandleGeneratePDF generates a formatted PDF table with watermark and returns its download URL.
func HandleGeneratePDF(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	headers, data, title, _, _ := ParseTableArgs(req)

	if len(headers) == 0 || len(data) == 0 {
		return client.ErrResult("both 'headers' and 'data' are required to generate a PDF table")
	}

	if client.Logger != nil {
		client.Logger.Printf("[HandleGeneratePDF] Generating PDF | title=%s | cols=%d | rows=%d", title, len(headers), len(data))
	}

	body := map[string]any{
		"title":   title,
		"headers": headers,
		"data":    data,
	}

	resp, err := client.DoRequest("POST", "/pdf", nil, body, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(resp)
}

// HandleGenerateCSV generates a CSV document from data rows and returns its download URL.
func HandleGenerateCSV(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	headers, data, title, filename, _ := ParseTableArgs(req)

	if len(headers) == 0 || len(data) == 0 {
		return client.ErrResult("both 'headers' and 'data' are required to generate a CSV export")
	}

	if client.Logger != nil {
		client.Logger.Printf("[HandleGenerateCSV] Generating CSV | filename=%s | title=%s | cols=%d | rows=%d", filename, title, len(headers), len(data))
	}

	body := map[string]any{
		"title":    title,
		"filename": filename,
		"headers":  headers,
		"data":     data,
	}

	resp, err := client.DoRequest("POST", "/csv", nil, body, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(resp)
}

// HandleGenerateExcel generates a styled Microsoft Excel (.xlsx) spreadsheet and returns its download URL.
func HandleGenerateExcel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	headers, data, title, filename, sheet := ParseTableArgs(req)

	if len(headers) == 0 || len(data) == 0 {
		return client.ErrResult("both 'headers' and 'data' are required to generate an Excel spreadsheet")
	}

	if client.Logger != nil {
		client.Logger.Printf("[HandleGenerateExcel] Generating Excel | filename=%s | sheet=%s | title=%s | cols=%d | rows=%d", filename, sheet, title, len(headers), len(data))
	}

	body := map[string]any{
		"title":    title,
		"filename": filename,
		"sheet":    sheet,
		"headers":  headers,
		"data":     data,
	}

	resp, err := client.DoRequest("POST", "/excel", nil, body, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(resp)
}
