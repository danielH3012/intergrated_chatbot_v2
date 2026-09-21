package controller

import (
	"context"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
)

func init() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
}

func TestParseIntentJSON_FileConversion(t *testing.T) {
	raw := `{"category": "file_conversion", "target_tool": "no_tools", "is_file_convert": true, "is_off_topic": true, "reason": "user asks to convert spreadsheet to pdf"}`
	res, err := parseIntentJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing intent JSON: %v", err)
	}
	if res.Category != "file_conversion" {
		t.Errorf("expected category file_conversion, got %s", res.Category)
	}
	if !res.IsFileConvert {
		t.Errorf("expected IsFileConvert to be true")
	}
	if !res.IsOffTopic {
		t.Errorf("expected IsOffTopic to be true")
	}
	if res.TargetTool != "no_tools" {
		t.Errorf("expected TargetTool no_tools, got %s", res.TargetTool)
	}
}

func TestParseIntentJSON_Unrelated(t *testing.T) {
	raw := `{"category": "unrelated", "target_tool": "no_tools", "is_off_topic": true, "reason": "asking for a lasagna recipe"}`
	res, err := parseIntentJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing intent JSON: %v", err)
	}
	if res.Category != "unrelated" {
		t.Errorf("expected category unrelated, got %s", res.Category)
	}
	if !res.IsOffTopic {
		t.Errorf("expected IsOffTopic to be true")
	}
	if res.TargetTool != "no_tools" {
		t.Errorf("expected TargetTool no_tools, got %s", res.TargetTool)
	}
}

func TestParseIntentJSON_GeneratePDF(t *testing.T) {
	raw := `{"category": "generate_pdf", "target_tool": "generate_pdf", "is_pdf_report": true, "reason": "export assets to PDF"}`
	res, err := parseIntentJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing intent JSON: %v", err)
	}
	if res.Category != "generate_pdf" {
		t.Errorf("expected category generate_pdf, got %s", res.Category)
	}
	if !res.IsPDFReport {
		t.Errorf("expected IsPDFReport to be true")
	}
	if res.IsFileConvert {
		t.Errorf("expected IsFileConvert to be false")
	}
	if res.IsOffTopic {
		t.Errorf("expected IsOffTopic to be false")
	}
	if res.TargetTool != "generate_pdf" {
		t.Errorf("expected TargetTool generate_pdf, got %s", res.TargetTool)
	}
}

func TestParseIntentJSON_GenerateCSV(t *testing.T) {
	raw := `{"category": "generate_pdf", "target_tool": "generate_csv", "is_pdf_report": true, "reason": "export assets to CSV"}`
	res, err := parseIntentJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing intent JSON: %v", err)
	}
	if res.Category != "generate_pdf" {
		t.Errorf("expected category generate_pdf, got %s", res.Category)
	}
	if !res.IsPDFReport {
		t.Errorf("expected IsPDFReport to be true")
	}
	if res.TargetTool != "generate_csv" {
		t.Errorf("expected TargetTool generate_csv, got %s", res.TargetTool)
	}
}

func TestParseIntentJSON_GenerateExcel(t *testing.T) {
	raw := `{"category": "generate_pdf", "target_tool": "generate_excel", "is_pdf_report": true, "reason": "export assets to Excel"}`
	res, err := parseIntentJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing intent JSON: %v", err)
	}
	if res.Category != "generate_pdf" {
		t.Errorf("expected category generate_pdf, got %s", res.Category)
	}
	if !res.IsPDFReport {
		t.Errorf("expected IsPDFReport to be true")
	}
	if res.TargetTool != "generate_excel" {
		t.Errorf("expected TargetTool generate_excel, got %s", res.TargetTool)
	}
}

func TestParseIntentJSON_DeleteAsset(t *testing.T) {
	raw := `{"category": "delete_asset", "reason": "delete asset AST-001"}`
	res, err := parseIntentJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing intent JSON: %v", err)
	}
	if res.Category != "delete_asset" {
		t.Errorf("expected category delete_asset, got %s", res.Category)
	}
	if !res.IsMutation {
		t.Errorf("expected IsMutation to be true")
	}
	if !res.IsDelete {
		t.Errorf("expected IsDelete to be true")
	}
	if res.TargetTool != "delete_asset" {
		t.Errorf("expected TargetTool delete_asset, got %s", res.TargetTool)
	}
}

func TestParseIntentJSON_UpdateAsset(t *testing.T) {
	raw := `{"category": "update_asset", "reason": "change laptop location"}`
	res, err := parseIntentJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing intent JSON: %v", err)
	}
	if res.Category != "update_asset" {
		t.Errorf("expected category update_asset, got %s", res.Category)
	}
	if !res.IsMutation {
		t.Errorf("expected IsMutation to be true")
	}
	if res.IsDelete {
		t.Errorf("expected IsDelete to be false")
	}
	if res.TargetTool != "update_asset" {
		t.Errorf("expected TargetTool update_asset, got %s", res.TargetTool)
	}
}

func TestParseIntentJSON_AddAsset(t *testing.T) {
	raw := `{"category": "add_asset", "reason": "register 5 monitors"}`
	res, err := parseIntentJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing intent JSON: %v", err)
	}
	if res.Category != "add_asset" {
		t.Errorf("expected category add_asset, got %s", res.Category)
	}
	if !res.IsMutation {
		t.Errorf("expected IsMutation to be true")
	}
	if res.TargetTool != "add_asset" {
		t.Errorf("expected TargetTool add_asset, got %s", res.TargetTool)
	}
}

func TestParseIntentJSON_QueryAsset(t *testing.T) {
	raw := `{"category": "query_asset", "reason": "count available laptops"}`
	res, err := parseIntentJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing intent JSON: %v", err)
	}
	if res.Category != "query_asset" {
		t.Errorf("expected category query_asset, got %s", res.Category)
	}
	if res.IsMutation {
		t.Errorf("expected IsMutation to be false")
	}
	if res.TargetTool != "get_assets" {
		t.Errorf("expected TargetTool get_assets, got %s", res.TargetTool)
	}
}

func TestParseIntentJSON_MarkdownCodeFence(t *testing.T) {
	raw := "```json\n{\n  \"category\": \"generate_pdf\",\n  \"target_tool\": \"generate_pdf\",\n  \"is_pdf_report\": true\n}\n```"
	res, err := parseIntentJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing markdown code fence JSON: %v", err)
	}
	if res.Category != "generate_pdf" || !res.IsPDFReport || res.TargetTool != "generate_pdf" {
		t.Errorf("unexpected parsed result from code fence: %+v", res)
	}
}

func TestParseIntentJSON_EmbeddedJSON(t *testing.T) {
	raw := "Here is my intent analysis:\n{\"category\": \"unrelated\", \"is_off_topic\": true}\nHope this helps!"
	res, err := parseIntentJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing embedded JSON: %v", err)
	}
	if res.Category != "unrelated" || !res.IsOffTopic || res.TargetTool != "no_tools" {
		t.Errorf("unexpected parsed result from embedded JSON: %+v", res)
	}
}

func TestParseIntentJSON_InvalidJSON(t *testing.T) {
	raw := "not a valid json string at all"
	_, err := parseIntentJSON(raw)
	if err == nil {
		t.Errorf("expected error for invalid JSON, got nil")
	}
}

func TestRoleBasedToolVisibility(t *testing.T) {
	allTools := []mcp.Tool{
		{Name: "get_assets"},
		{Name: "add_asset"},
		{Name: "update_asset"},
		{Name: "delete_asset"},
		{Name: "generate_pdf"},
		{Name: "create_schedule"},
		{Name: "get_schedules"},
		{Name: "no_tools"},
	}

	// Viewer tools
	viewerTools := filterRoles(allTools, "viewer")
	if hasTool(viewerTools, "add_asset") {
		t.Errorf("viewer should NOT have add_asset")
	}
	if hasTool(viewerTools, "update_asset") {
		t.Errorf("viewer should NOT have update_asset")
	}
	if hasTool(viewerTools, "delete_asset") {
		t.Errorf("viewer should NOT have delete_asset")
	}
	if hasTool(viewerTools, "create_schedule") {
		t.Errorf("viewer should NOT have create_schedule")
	}
	if !hasTool(viewerTools, "get_assets") {
		t.Errorf("viewer should have get_assets")
	}
	if !hasTool(viewerTools, "get_schedules") {
		t.Errorf("viewer should have get_schedules")
	}
	if !hasTool(viewerTools, "generate_pdf") {
		t.Errorf("viewer should have generate_pdf")
	}

	// Operator tools
	operatorTools := filterRoles(allTools, "operator")
	if hasTool(operatorTools, "delete_asset") {
		t.Errorf("operator should NOT have delete_asset")
	}
	if !hasTool(operatorTools, "add_asset") {
		t.Errorf("operator should have add_asset")
	}
	if !hasTool(operatorTools, "update_asset") {
		t.Errorf("operator should have update_asset")
	}
	if !hasTool(operatorTools, "get_assets") {
		t.Errorf("operator should have get_assets")
	}
	if !hasTool(operatorTools, "create_schedule") {
		t.Errorf("operator should have create_schedule")
	}
	if !hasTool(operatorTools, "get_schedules") {
		t.Errorf("operator should have get_schedules")
	}

	// Admin tools
	adminTools := filterRoles(allTools, "admin")
	if !hasTool(adminTools, "delete_asset") {
		t.Errorf("admin should have delete_asset")
	}
	if !hasTool(adminTools, "add_asset") {
		t.Errorf("admin should have add_asset")
	}
	if !hasTool(adminTools, "create_schedule") {
		t.Errorf("admin should have create_schedule")
	}
	if !hasTool(adminTools, "get_schedules") {
		t.Errorf("admin should have get_schedules")
	}

	// Dynamic intent.TargetTool capability check (unhardcoded)
	updateIntent := &IntentResult{Category: "update_asset", TargetTool: "update_asset", IsMutation: true}
	deleteIntent := &IntentResult{Category: "delete_asset", TargetTool: "delete_asset", IsMutation: true, IsDelete: true}
	queryIntent := &IntentResult{Category: "query_asset", TargetTool: "get_assets"}
	scheduleIntent := &IntentResult{Category: "create_schedule", TargetTool: "create_schedule", IsMutation: true}

	if !hasTool(operatorTools, updateIntent.TargetTool) {
		t.Errorf("operator should be permitted for update_asset intent")
	}
	if hasTool(operatorTools, deleteIntent.TargetTool) {
		t.Errorf("operator should NOT be permitted for delete_asset intent")
	}
	if !hasTool(operatorTools, queryIntent.TargetTool) {
		t.Errorf("operator should be permitted for query_asset intent")
	}
	if !hasTool(operatorTools, scheduleIntent.TargetTool) {
		t.Errorf("operator should be permitted for create_schedule intent")
	}
	if hasTool(viewerTools, scheduleIntent.TargetTool) {
		t.Errorf("viewer should NOT be permitted for create_schedule intent")
	}
	if !hasTool(adminTools, scheduleIntent.TargetTool) {
		t.Errorf("admin should be permitted for create_schedule intent")
	}
	if hasTool(viewerTools, updateIntent.TargetTool) {
		t.Errorf("viewer should NOT be permitted for update_asset intent")
	}
	if !hasTool(adminTools, deleteIntent.TargetTool) {
		t.Errorf("admin should be permitted for delete_asset intent")
	}
}

func TestParseIntentJSON_FileConversionVariants(t *testing.T) {
	cases := []string{
		`{"category": "file_conversion", "target_tool": "no_tools", "is_file_convert": true, "is_off_topic": true, "reason": "user asks to convert excel to pdf"}`,
		`{"category": "file_conversion", "target_tool": "no_tools", "is_file_convert": true, "is_off_topic": true, "reason": "user asks to convert file to csv"}`,
		`{"category": "file_conversion", "target_tool": "no_tools", "is_file_convert": true, "is_off_topic": true, "reason": "user asks to convert file to excel"}`,
	}
	for _, raw := range cases {
		res, err := parseIntentJSON(raw)
		if err != nil {
			t.Fatalf("unexpected error parsing intent JSON: %v", err)
		}
		if res.Category != "file_conversion" {
			t.Errorf("expected category file_conversion, got %s", res.Category)
		}
		if !res.IsFileConvert {
			t.Errorf("expected IsFileConvert to be true")
		}
		if !res.IsOffTopic {
			t.Errorf("expected IsOffTopic to be true")
		}
		if res.TargetTool != "no_tools" {
			t.Errorf("expected TargetTool no_tools, got %s", res.TargetTool)
		}
	}
}

func TestParseIntentJSON_Schedule(t *testing.T) {
	createRaw := `{"category": "create_schedule", "target_tool": "create_schedule", "is_mutation": true, "reason": "schedule audit"}`
	res, err := parseIntentJSON(createRaw)
	if err != nil {
		t.Fatalf("unexpected error parsing intent JSON: %v", err)
	}
	if res.Category != "create_schedule" || res.TargetTool != "create_schedule" || !res.IsMutation || res.IsOffTopic {
		t.Errorf("expected create_schedule mutation, got %+v", res)
	}

	getRaw := `{"category": "get_schedules", "target_tool": "get_schedules", "is_mutation": false, "reason": "view schedules"}`
	resGet, err := parseIntentJSON(getRaw)
	if err != nil {
		t.Fatalf("unexpected error parsing intent JSON: %v", err)
	}
	if resGet.Category != "get_schedules" || resGet.TargetTool != "get_schedules" || resGet.IsMutation || resGet.IsOffTopic {
		t.Errorf("expected get_schedules non-mutation, got %+v", resGet)
	}
}

func TestLiveSearchIntent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live LLM test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Unrelated test
	resUnrelated := SearchIntent(ctx, "Tolong berikan saya resep nasi goreng kambing yang enak", nil, "")
	if resUnrelated.Category != "unrelated" && !resUnrelated.IsOffTopic {
		t.Errorf("expected unrelated/off-topic intent, got %+v", resUnrelated)
	}

	// 2. Prohibited file conversion test
	resConv := SearchIntent(ctx, "Tolong konversi file excel ini ke format pdf", map[string]any{"attachment_name": "data.xlsx"}, "")
	if resConv.Category != "file_conversion" && !resConv.IsFileConvert {
		t.Errorf("expected file_conversion intent, got %+v", resConv)
	}

	// 3. Generate PDF report test
	resPDF := SearchIntent(ctx, "Buatkan laporan PDF daftar aset laptop kantor", nil, "")
	if resPDF.Category != "generate_pdf" && !resPDF.IsPDFReport {
		t.Errorf("expected generate_pdf intent, got %+v", resPDF)
	}

	// 4. Update asset test
	resUpdate := SearchIntent(ctx, "update termostat price to 150000", nil, "")
	t.Logf("Result for 'update termostat price to 150000': %+v", resUpdate)
	if resUpdate.Category != "update_asset" && !resUpdate.IsMutation {
		t.Errorf("expected update_asset intent, got %+v", resUpdate)
	}

	// 5. Audit schedule test (MUST NOT be off-topic and must map to create_schedule)
	resAudit := SearchIntent(ctx, "tolong buat jadwal audit aset kantor", nil, "")
	t.Logf("Result for 'tolong buat jadwal audit aset kantor': %+v", resAudit)
	if resAudit.IsOffTopic || resAudit.Category == "unrelated" {
		t.Errorf("expected audit schedule to NOT be off-topic, got %+v", resAudit)
	}
	if resAudit.Category != "create_schedule" || resAudit.TargetTool != "create_schedule" {
		t.Errorf("expected create_schedule category and tool, got %+v", resAudit)
	}
}

func TestIsAuditScheduleQuery(t *testing.T) {
	validQueries := []string{
		"buat jadwal audit",
		"buatkan jadwal audit",
		"jadwal audit aset",
		"tolong jadwalkan audit untuk laptop kantor",
		"rencana audit berkala",
		"jadwal stock opname inventaris",
		"kapan jadwal audit barang?",
	}
	for _, q := range validQueries {
		if !isAuditScheduleQuery(q) {
			t.Errorf("expected %q to be recognized as audit schedule query", q)
		}
	}

	invalidQueries := []string{
		"resep nasi goreng",
		"buatkan saya puisi",
		"hitung 10 + 5",
	}
	for _, q := range invalidQueries {
		if isAuditScheduleQuery(q) {
			t.Errorf("expected %q to NOT be recognized as audit schedule query", q)
		}
	}
}

func TestTryParseToolCalls_DeepSeekDSML(t *testing.T) {
	raw := `<tool_call>
{"name": "generate_csv", "arguments": {"title": "Laporan Aset Minggu Ini - qtera mandiri", "headers": ["No", "Asset ID", "Nama Aset"], "data": [["1", "AST-57BDAD55", "Site Warranty"]], "filename": "asset_report_this_week.csv"}}</|DSML|>
< | | DSML | | tool_call>`

	calls := tryParseToolCalls(raw)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Name != "generate_csv" {
		t.Errorf("expected generate_csv, got %s", calls[0].Name)
	}
	if title, ok := calls[0].Arguments["title"].(string); !ok || title != "Laporan Aset Minggu Ini - qtera mandiri" {
		t.Errorf("unexpected title: %v", calls[0].Arguments["title"])
	}
}

func TestTryParseToolCalls_QwenPlugin(t *testing.T) {
	raw := `<|action_start|><|plugin|>
{"name": "get_assets", "parameters": {"search": "laptop"}}
<|action_end|>`

	calls := tryParseToolCalls(raw)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Name != "get_assets" {
		t.Errorf("expected get_assets, got %s", calls[0].Name)
	}
	if search, ok := calls[0].Arguments["search"].(string); !ok || search != "laptop" {
		t.Errorf("unexpected search param: %v", calls[0].Arguments["search"])
	}
}

func TestTryParseToolCalls_ClaudeToolUse(t *testing.T) {
	raw := `<tool_use>
{"name": "generate_excel", "input": {"title": "Laporan Excel", "headers": ["ID", "Nama"], "data": [["AST-1", "Monitor"]]}}
</tool_use>`

	calls := tryParseToolCalls(raw)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Name != "generate_excel" {
		t.Errorf("expected generate_excel, got %s", calls[0].Name)
	}
	if title, ok := calls[0].Arguments["title"].(string); !ok || title != "Laporan Excel" {
		t.Errorf("unexpected title: %v", calls[0].Arguments["title"])
	}
}

func TestTryParseToolCalls_OpenAIHermes(t *testing.T) {
	raw := `<|tool_call_start|>
{"name": "update_asset", "arguments": "{\"id\": \"AST-12345\", \"price\": \"500\"}"}
<|tool_call_end|>`

	calls := tryParseToolCalls(raw)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Name != "update_asset" {
		t.Errorf("expected update_asset, got %s", calls[0].Name)
	}
	if id, ok := calls[0].Arguments["id"].(string); !ok || id != "AST-12345" {
		t.Errorf("unexpected id: %v", calls[0].Arguments["id"])
	}
}

func TestTryParseToolCalls_UnclosedTag(t *testing.T) {
	raw := `<tool_call>
{"name": "generate_pdf", "arguments": {"title": "PDF Laporan"}}`

	calls := tryParseToolCalls(raw)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call from unclosed tag, got %d", len(calls))
	}
	if calls[0].Name != "generate_pdf" {
		t.Errorf("expected generate_pdf, got %s", calls[0].Name)
	}
}

func TestSanitizeModelReply_AllModels(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "DeepSeek DSML with markup",
			input: `<tool_call>
{"name": "generate_csv", "arguments": {"title": "Laporan"}}</|DSML|>
< | | DSML | | tool_call>`,
			expected: "",
		},
		{
			name: "Reasoning think tags and tool call",
			input: `<think>
I must query the assets first.
</think>
<tool_call>
{"name": "get_assets", "arguments": {}}
</tool_call>`,
			expected: "",
		},
		{
			name: "Conversational text with tool call",
			input: `Here is your requested report:
<tool_call>
{"name": "generate_csv", "arguments": {"title": "Laporan"}}
</tool_call>`,
			expected: "Here is your requested report:",
		},
		{
			name: "Qwen plugin tags",
			input: `<|action_start|><|plugin|>
{"name": "get_assets", "parameters": {}}
<|action_end|>`,
			expected: "",
		},
		{
			name: "OpenAI tool call start tags",
			input: `<|tool_call_start|>
{"name": "get_assets", "arguments": {}}
<|tool_call_end|>`,
			expected: "",
		},
		{
			name: "Claude tool use tags",
			input: `<tool_use>
{"name": "generate_excel", "input": {}}
</tool_use>`,
			expected: "",
		},
		{
			name:     "Markdown fenced json tool call",
			input:    "```json\n{\"name\": \"delete_asset\", \"arguments\": {\"id\": \"AST-99\"}}\n```",
			expected: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeModelReply(tc.input)
			if got != tc.expected {
				t.Errorf("SanitizeModelReply(%q) = %q, expected %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestNewServerConnection(t *testing.T) {
	c, err := NewServerConnection()
	if err != nil {
		t.Fatalf("failed to connect to MCP server: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	toolsResp, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("failed to list tools from MCP server: %v", err)
	}

	if len(toolsResp.Tools) == 0 {
		t.Fatalf("expected tools from MCP server, got 0")
	}

	t.Logf("Successfully connected to MCP server and loaded %d tools!", len(toolsResp.Tools))
}


