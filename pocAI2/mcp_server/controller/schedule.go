package controller

import (
	"context"
	"net/url"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp_server/client"
)

// RegisterScheduleTools mounts schedule-related tools onto the MCP server.
func RegisterScheduleTools(s *server.MCPServer) {
	// 1. create_schedule: insert record to public.schedule
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
	), HandleCreateSchedule)

	// 2. get_schedules: list schedules from public.schedule
	s.AddTool(mcp.NewTool("get_schedules",
		mcp.WithDescription(
			`List or view existing schedules from the public.schedule database table.
Use this tool whenever the user asks to check, view, or list existing audit schedules or tasks.
Parameters:
  - category: Optional category keyword filter`),
		mcp.WithString("category", mcp.Description("Optional category filter")),
	), HandleGetSchedules)
}

// HandleCreateSchedule inserts a new schedule record into public.schedule.
func HandleCreateSchedule(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)

	category := client.GetArgString(req, "category", "nama_jadwal", "title", "name")
	start := client.GetArgString(req, "start", "start_date", "tanggal_mulai", "tanggal")
	frequency := client.GetArgString(req, "frequency", "frekuensi", "periode")

	if category == "" {
		return client.ErrResult("category is required and cannot be empty")
	}
	if start == "" {
		start = time.Now().Format("2006-01-02")
	}
	if frequency == "" {
		frequency = "Once"
	}

	if client.Logger != nil {
		client.Logger.Printf("[HandleCreateSchedule] Inserting schedule | category=%s | start=%s | frequency=%s", category, start, frequency)
	}

	body := map[string]any{
		"category":  category,
		"start":     start,
		"frequency": frequency,
	}

	resp, err := client.DoRequest("POST", "/schedule", nil, body, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(resp)
}

// HandleGetSchedules retrieves schedule records from public.schedule.
func HandleGetSchedules(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	creds := client.ExtractCredentials(req)
	params := url.Values{}
	if category := client.GetArgString(req, "category", "search"); category != "" {
		params.Set("category", category)
	}

	data, err := client.DoRequest("GET", "/schedule", params, nil, creds)
	if err != nil {
		return client.ErrResult(err.Error())
	}
	return client.ToolResult(data)
}
