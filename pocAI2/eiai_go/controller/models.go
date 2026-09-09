package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	GeneratorModel1 = "ox-alpha-free"
	GeneratorModel2 = "glm-5.3-flash"
	GeneratorModel3 = "mimo-v2.5"
	GeneratorModel4 = "gpt-5.6-luna"
	GeneratorModel5 = "qwen3.8-flash"
	GeneratorModel6 = "deepseek-v4-flash"
	GeneratorModel7 = "liquid/lfm-2.5-2.6b:free"
	GeneratorModel8 = "z-ai/glm-5.2:free"
)

var GeneratorModel = GeneratorModel6

// AgentResult represents the final structured response from CallTools.
type AgentResult struct {
	Answer     string          `json:"answer"`
	Attachment *AttachmentInfo `json:"attachment,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResult struct {
	RawOutput        string `json:"raw_output"`
	InputTokens      int    `json:"input_tokens"`
	OutputTokens     int    `json:"output_tokens"`
	ReasoningDetails any    `json:"reasoning_details,omitempty"`
}

// ChatGenerate calls the OpenRouter/OpenAI chat completion API with retry & backoff.
func ChatGenerate(ctx context.Context, messages []Message, tools any, maxNewTokens int, model string, retries int) (*ChatResult, error) {
	_ = godotenv.Load(".env")

	if model == "" {
		model = GeneratorModel
	}
	if maxNewTokens <= 0 {
		maxNewTokens = 4096
	}

	baseURL := os.Getenv("BASE_AI_URL")
	if strings.Contains(baseURL, "openrouter.ai") && !strings.Contains(model, "/") {
		log.Printf("[chat_generate] Model '%s' lacks vendor prefix for OpenRouter, defaulting to '%s'", model, GeneratorModel7)
		model = GeneratorModel7
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"

	reqBody := map[string]any{
		"model":      model,
		"messages":   messages,
		"max_tokens": maxNewTokens,
		"extra_body": map[string]any{"reasoning": map[string]any{"enabled": true}},
	}
	if tools != nil {
		reqBody["tools"] = tools
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 90 * time.Second}
	var lastErr error

	for attempt := 0; attempt <= retries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+os.Getenv("AI_KEY"))

		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()

			var res struct {
				Choices []struct {
					Message struct {
						Content          string `json:"content"`
						ReasoningContent string `json:"reasoning_content"`
						ReasoningDetails any    `json:"reasoning_details"`
						Reasoning        any    `json:"reasoning"`
					} `json:"message"`
				} `json:"choices"`
				Usage struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
				} `json:"usage"`
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}

			if decodeErr := json.NewDecoder(resp.Body).Decode(&res); decodeErr == nil {
				if resp.StatusCode == http.StatusOK {
					var rawOutput string
					var reasoning any

					if len(res.Choices) > 0 {
						rawOutput = res.Choices[0].Message.Content
						if strings.TrimSpace(rawOutput) == "" && strings.TrimSpace(res.Choices[0].Message.ReasoningContent) != "" {
							rawOutput = res.Choices[0].Message.ReasoningContent
						}
						reasoning = res.Choices[0].Message.ReasoningDetails
						if reasoning == nil {
							reasoning = res.Choices[0].Message.Reasoning
						}
						if reasoning == nil {
							reasoning = res.Choices[0].Message.ReasoningContent
						}
					}

					return &ChatResult{
						RawOutput:        rawOutput,
						InputTokens:      res.Usage.PromptTokens,
						OutputTokens:     res.Usage.CompletionTokens,
						ReasoningDetails: reasoning,
					}, nil
				}
				err = fmt.Errorf("api error (%d): %s", resp.StatusCode, res.Error.Message)
			} else {
				err = decodeErr
			}
		}

		lastErr = err
		wait := computeRetryWait(resp, attempt)

		if attempt < retries {
			log.Printf("[chat_generate] attempt %d/%d failed (%v), retrying in %v", attempt+1, retries+1, err, wait)
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		} else {
			log.Printf("[chat_generate] all %d attempts failed: %v", retries+1, err)
		}
	}

	return nil, lastErr
}

func computeRetryWait(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		if sec, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return time.Duration(1<<attempt) * time.Second
}

func getTemporalContextPrompt() string {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -(weekday - 1))

	dateReference := fmt.Sprintf(
		"Current Date Reference:\n"+
			"- Today: %s\n"+
			"- Start of Current Week (Monday): %s\n"+
			"- Current Month: %s (matches date prefix '%s')\n"+
			"- Current Year: %s\n"+
			"- Start of Current Month: %s\n"+
			"- Current ISO Timestamp: %s",
		now.Format("2006-01-02 (Monday)"),
		startOfWeek.Format("2006-01-02"),
		now.Format("January 2006"),
		now.Format("2006-01"),
		now.Format("2006"),
		now.Format("2006-01-01"),
		now.Format(time.RFC3339),
	)
	return dateReference
}

// extractContext inspects accumulated API tool results, performs analytical reasoning & filtering with the LLM,
// determines whether data is sufficient (<verdict status="ENOUGH"> vs <verdict status="NEED_MORE">),
// and returns (isEnough bool, extractedFacts string, missingInfo string).
func extractContext(ctx context.Context, query string, intent *IntentResult, accumulatedResults []ToolExecutionResult, userContext map[string]any, iteration int, maxIterations int, model string) (bool, string, string) {
	var history []ChatMessage
	if userContext != nil {
		if h, ok := userContext["history"].([]ChatMessage); ok {
			history = h
		}
	}
	historyStr := FormatChatHistoryForLlm(history)
	hasAttachment := false
	if userContext != nil {
		if attachText, ok := userContext["attachment_text"].(string); ok && attachText != "" {
			hasAttachment = true
		}
	}

	onlyNoTools := len(accumulatedResults) > 0
	for _, res := range accumulatedResults {
		if res.Tool != "no_tools" && res.Tool != "no_tool" {
			onlyNoTools = false
			break
		}
	}

	if (len(accumulatedResults) == 0 || onlyNoTools) && historyStr == "" && !hasAttachment {
		return true, "No records found from API tool.", ""
	}

	if model == "" {
		model = GeneratorModel
	}

	var rawDataStr string
	if len(accumulatedResults) == 0 || onlyNoTools {
		rawDataStr = "No external database tool executed (greeting or non-asset inquiry)."
	} else {
		rawDataStr = formatMultiRawDataForLlm(accumulatedResults)
	}
	temporalInfo := getTemporalContextPrompt()
	companyInfo := ""
	if userContext != nil {
		if comp, ok := userContext["company"].(string); ok && comp != "" {
			companyInfo = fmt.Sprintf(" The requesting user belongs to company '%s'.", comp)
		} else if compInfo, ok := userContext["company_info"].(map[string]any); ok {
			if compName, ok := compInfo["company_name"].(string); ok && compName != "" {
				companyInfo = fmt.Sprintf(" The requesting user belongs to company '%s'.", compName)
			}
		}
	}

	systemPrompt := fmt.Sprintf(
		"You are an analytical data extraction, context filtering, and reasoning engine for QTERA IT Asset Management.%s\n\n"+
			"%s\n\n"+
			"Your role is to analyze the user's question across all available context:\n"+
			"1. Data returned from internal API tools (assets catalog)\n"+
			"2. Recent Conversation History (chat history)\n"+
			"3. Attached documents (if provided)\n\n"+
			"You must perform all necessary analytical reasoning, date filtering, counting, filtering of chat history, and context extraction, "+
			"and determine whether additional data is needed to answer the question.\n\n"+
			"Instructions & Rules:\n"+
			"1. **Sufficiency Evaluation (Line 1)**:\n"+
			"   - On the VERY FIRST LINE of your response, output a verdict tag:\n"+
			"     * If crucial information is still missing from the API data to answer what the user asked: `<verdict status=\"NEED_MORE\" missing=\"specific missing detail\"/>`\n"+
			"     * If and ONLY if the gathered data and context contain the actual answers to the user's specific request: `<verdict status=\"ENOUGH\"/>`\n"+
			"     * If the user's question is unrelated to IT Asset Management or equipment inventory: `<verdict status=\"ENOUGH\"/>`\n"+
			"2. **Chat History Filtering & Distillation (Pilahlah Riwayat Percakapan)**:\n"+
			"   - Carefully inspect the Recent Conversation History.\n"+
			"   - Filter and extract ONLY the specific facts, previous asset comparisons, IDs, file analyses, or context relevant to the user's current question.\n"+
			"   - Exclude unrelated pleasantries, irrelevant queries, or outdated chatter.\n"+
			"3. **Temporal & Date Reasoning**:\n"+
			"   - Use the Current Date Reference above to resolve relative time terms (e.g. 'this month', 'today', 'this year', 'last week').\n"+
			"   - Compare records' purchase date or created_at fields against the target period (e.g. for 'this month', match the current YYYY-MM prefix).\n"+
			"   - Only include records matching the requested time window.\n"+
			"4. **Counting & Aggregations ('how many', 'total count', 'summary')**:\n"+
			"   - When the user asks 'how many' or for a count/summary, explicitly calculate and state the total count of matching assets (e.g. 'Total matching assets: X').\n"+
			"   - Include details of the matching assets (name, category, brand, model, price, location) so the count can be verified.\n"+
			"5. **Filtering by Category, Brand, Location, Status, Company**:\n"+
			"   - Check field values accurately.\n"+
			"6. **Factual Accuracy & Completeness**:\n"+
			"   - If no records match the criteria/filters, explicitly state that no matching records were found.\n"+
			"   - Do NOT invent, assume, or hallucinate records not in the data.\n"+
			"7. **Live Data Precedence & Freshness**:\n"+
			"   - Live data from API tools (`Accumulated API Data`) is the single source of truth and strictly supersedes any past statements in `Recent Conversation History`.\n"+
			"8. **Output Format**:\n"+
			"   - Following the verdict tag, present the distilled and extracted facts clearly (combining verified live data, filtered conversation history facts, and document context).",
		companyInfo,
		temporalInfo,
	)

	var userPromptBuilder strings.Builder
	if historyStr != "" {
		userPromptBuilder.WriteString(fmt.Sprintf("=== RECENT CONVERSATION HISTORY (RAW) ===\n%s\n\n", historyStr))
	}
	if userContext != nil {
		if attachText, ok := userContext["attachment_text"].(string); ok && attachText != "" {
			attachName := "Attached Document"
			if name, ok := userContext["attachment_name"].(string); ok && name != "" {
				attachName = name
			}
			userPromptBuilder.WriteString(fmt.Sprintf("=== ATTACHED DOCUMENT CONTEXT (%s) ===\n%s\n\n", attachName, attachText))
		}
	}
	userPromptBuilder.WriteString(fmt.Sprintf("=== LIVE DATABASE DATA ===\n%s\n\n=== USER QUESTION ===\n%s", rawDataStr, query))

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPromptBuilder.String()},
	}

	res, err := ChatGenerate(ctx, messages, nil, 32768, model, 3)
	if err != nil {
		log.Printf("[extractContext] Failed to extract context with LLM: %v", err)
		return true, rawDataStr, ""
	}
	log.Printf("[extractContext] Iteration %d tokens -> input: %d, output: %d, total: %d", iteration, res.InputTokens, res.OutputTokens, res.InputTokens+res.OutputTokens)

	rawExtraction := strings.TrimSpace(res.RawOutput)
	if rawExtraction == "" {
		rawExtraction = rawDataStr
	}

	// Parse verdict tag
	verdictRegex := regexp.MustCompile(`(?i)<verdict\s+status=["'](ENOUGH|NEED_MORE)["'](?:\s+missing=["'](.*?)["'])?`)
	isEnough := true
	missingInfo := ""

	if match := verdictRegex.FindStringSubmatch(rawExtraction); len(match) >= 2 {
		status := strings.ToUpper(match[1])
		if len(match) >= 3 {
			missingInfo = strings.TrimSpace(match[2])
		}
		if status == "NEED_MORE" && iteration < maxIterations {
			isEnough = false
		}
	}

	// 2. Never allow extractContext to claim ENOUGH if target tool has not executed yet
	if intent == nil {
		intent = searchIntent(ctx, query, userContext, model)
	}
	hasExecutedTarget := false
	for _, res := range accumulatedResults {
		if res.Tool == intent.TargetTool {
			hasExecutedTarget = true
			break
		}
	}
	if (intent.IsMutation || intent.IsPDFReport) && !hasExecutedTarget && iteration < maxIterations {
		isEnough = false
		if missingInfo == "" {
			if intent.IsPDFReport {
				target := intent.TargetTool
				if target == "" {
					target = "generate_pdf"
				}
				missingInfo = fmt.Sprintf("Asset data has been gathered. You must now call the '%s' tool to export the assets into a downloadable file.", target)
			} else {
				missingInfo = fmt.Sprintf("The requested action '%s' has not been executed yet. Proceed to call '%s'.", intent.Category, intent.TargetTool)
			}
		}
	}

	tagStripRegex := regexp.MustCompile(`(?i)<verdict\s+.*?/>`)
	extractedFacts := strings.TrimSpace(tagStripRegex.ReplaceAllString(rawExtraction, ""))

	if extractedFacts == "" {
		extractedFacts = rawDataStr
	}

	return isEnough, extractedFacts, missingInfo
}

// IntentResult represents the semantic intent classified by the searchIntent agent function.
type IntentResult struct {
	Category      string `json:"category"`
	TargetTool    string `json:"target_tool"`
	IsMutation    bool   `json:"is_mutation"`
	IsDelete      bool   `json:"is_delete"`
	IsPDFReport   bool   `json:"is_pdf_report"`
	IsFileConvert bool   `json:"is_file_convert"`
	IsOffTopic    bool   `json:"is_off_topic"`
	Reason        string `json:"reason,omitempty"`
}

// parseIntentJSON parses and normalizes the JSON output produced by the LLM intent classifier.
func parseIntentJSON(raw string) (*IntentResult, error) {
	clean := strings.TrimSpace(raw)
	// Strip markdown code fences if present
	reFence := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	if match := reFence.FindStringSubmatch(clean); len(match) > 1 {
		clean = strings.TrimSpace(match[1])
	} else {
		// Strip leading unclosed code fence if present
		reLead := regexp.MustCompile("(?s)^```(?:json)?\\s*")
		clean = strings.TrimSpace(reLead.ReplaceAllString(clean, ""))
		// Find first '{' and last '}'
		start := strings.Index(clean, "{")
		end := strings.LastIndex(clean, "}")
		if start >= 0 && end > start {
			clean = clean[start : end+1]
		}
	}

	var result IntentResult
	if err := json.Unmarshal([]byte(clean), &result); err != nil {
		if calls := tryParseToolCalls(raw); len(calls) > 0 {
			switch calls[0].Name {
			case "add_asset":
				return &IntentResult{Category: "add_asset", TargetTool: "add_asset", IsMutation: true}, nil
			case "update_asset":
				return &IntentResult{Category: "update_asset", TargetTool: "update_asset", IsMutation: true}, nil
			case "delete_asset":
				return &IntentResult{Category: "delete_asset", TargetTool: "delete_asset", IsMutation: true, IsDelete: true}, nil
			case "generate_pdf":
				return &IntentResult{Category: "generate_pdf", TargetTool: "generate_pdf", IsPDFReport: true}, nil
			case "generate_csv":
				return &IntentResult{Category: "generate_pdf", TargetTool: "generate_csv", IsPDFReport: true}, nil
			case "generate_excel":
				return &IntentResult{Category: "generate_pdf", TargetTool: "generate_excel", IsPDFReport: true}, nil
			case "get_assets":
				return &IntentResult{Category: "query_asset", TargetTool: "get_assets"}, nil
			case "create_schedule", "add_schedule":
				return &IntentResult{Category: "create_schedule", TargetTool: "create_schedule", IsMutation: true}, nil
			case "get_schedules":
				return &IntentResult{Category: "get_schedules", TargetTool: "get_schedules"}, nil
			case "no_tools", "no_tool":
				return &IntentResult{Category: "greeting", TargetTool: "no_tools"}, nil
			}
		}
		return nil, fmt.Errorf("failed to unmarshal intent JSON: %w (raw: %s)", err, raw)
	}

	// Normalize flags based on category to guarantee consistent internal routing
	switch strings.ToLower(strings.TrimSpace(result.Category)) {
	case "file_conversion":
		result.Category = "file_conversion"
		result.IsFileConvert = true
		result.IsOffTopic = true
		result.TargetTool = "no_tools"
	case "unrelated":
		result.Category = "unrelated"
		result.IsOffTopic = true
		result.TargetTool = "no_tools"
	case "delete_asset":
		result.Category = "delete_asset"
		result.IsMutation = true
		result.IsDelete = true
		if result.TargetTool == "" {
			result.TargetTool = "delete_asset"
		}
	case "update_asset":
		result.Category = "update_asset"
		result.IsMutation = true
		if result.TargetTool == "" {
			result.TargetTool = "update_asset"
		}
	case "add_asset":
		result.Category = "add_asset"
		result.IsMutation = true
		if result.TargetTool == "" {
			result.TargetTool = "add_asset"
		}
	case "generate_pdf", "generate_csv", "generate_excel":
		result.Category = "generate_pdf"
		result.IsPDFReport = true
		if result.TargetTool == "" {
			lower := strings.ToLower(clean)
			if strings.Contains(lower, "excel") || strings.Contains(lower, "xlsx") {
				result.TargetTool = "generate_excel"
			} else if strings.Contains(lower, "csv") {
				result.TargetTool = "generate_csv"
			} else {
				result.TargetTool = "generate_pdf"
			}
		}
	case "query_asset":
		result.Category = "query_asset"
		if result.TargetTool == "" {
			result.TargetTool = "get_assets"
		}
	case "create_schedule", "add_schedule":
		result.Category = "create_schedule"
		result.IsMutation = true
		if result.TargetTool == "" {
			result.TargetTool = "create_schedule"
		}
	case "get_schedules":
		result.Category = "get_schedules"
		if result.TargetTool == "" {
			result.TargetTool = "get_schedules"
		}
	case "greeting":
		result.Category = "greeting"
		result.TargetTool = "no_tools"
	}

	return &result, nil
}

// searchIntent is an LLM agent function that semantically classifies user intent.
func searchIntent(ctx context.Context, query string, userContext map[string]any, model string) *IntentResult {
	if model == "" {
		model = GeneratorModel
	}

	systemInstruction := `You are an Intent Classification Agent for QTERA IT Asset Management.
Analyze the user query and any attached document context to classify the user's precise intent.

You must respond with ONLY a valid JSON object matching this schema:
{
  "category": "query_asset" | "add_asset" | "update_asset" | "delete_asset" | "generate_pdf" | "create_schedule" | "get_schedules" | "file_conversion" | "greeting" | "unrelated",
  "target_tool": "get_assets" | "add_asset" | "update_asset" | "delete_asset" | "generate_pdf" | "generate_csv" | "generate_excel" | "create_schedule" | "get_schedules" | "no_tools",
  "is_mutation": boolean,
  "is_delete": boolean,
  "is_pdf_report": boolean,
  "is_file_convert": boolean,
  "is_off_topic": boolean,
  "reason": "short explanation"
}

INTENT CATEGORIES AND RULES:
1. "file_conversion": The user explicitly uploaded/attached their own file in the current turn and asks to convert, transform, reformat, or translate that specific uploaded file into another file format (e.g., "convert this file to pdf", "jadikan file ini excel", "ubah dokumen terlampir ini ke csv", "convert excel to pdf", "convert pdf to excel").
   CRITICAL DISTINCTION:
   - "file_conversion" ONLY applies when the user uploaded their own file and asks to convert or reformat that file directly without database queries.
   - Requesting asset data or reports in another format (e.g. "can you also make it in excel?", "give me this week report in excel", "buatkan dalam excel", "export laporan ini ke csv/excel/pdf", "tampilkan dalam bentuk excel") is ALWAYS "generate_pdf" (target_tool="generate_excel" or "generate_csv" or "generate_pdf") from database records, NEVER "file_conversion".
   -> If and only if the user explicitly asks to convert an uploaded file: category="file_conversion", target_tool="no_tools", is_file_convert=true, is_off_topic=true.
2. "unrelated": Any question, coding task, creative writing, cooking recipe, math, or conversation unrelated to IT Asset Management.
   -> category="unrelated", target_tool="no_tools", is_off_topic=true.
   CRITICAL SCOPE RULE - ASSET AUDIT SCHEDULING IS NOT OFF-TOPIC:
   - Creating, planning, organizing, or viewing an IT asset audit schedule or maintenance task (e.g., "buat jadwal audit", "buatkan jadwal audit", "jadwal audit aset", "jadwalkan audit laptop/perangkat", "schedule asset audit", "lihat jadwal audit") is an essential, core activity of IT Asset Management and is STRICTLY NOT OFF-TOPIC (is_off_topic=false)!
   - For creating/planning audit schedule: category="create_schedule", target_tool="create_schedule", is_mutation=true, is_off_topic=false.
   - For viewing/listing audit schedule: category="get_schedules", target_tool="get_schedules", is_mutation=false, is_off_topic=false.
3. "generate_pdf": User asks to export, generate, create, or download an inventory/asset summary report or table in PDF, CSV, or Excel format from database records (e.g., "give me this week report in excel", "can you also make it in excel?", "buatkan laporan aset minggu ini dalam excel", "export aset ke csv", "laporan pdf").
   - For PDF report: category="generate_pdf", target_tool="generate_pdf", is_pdf_report=true.
   - For CSV report: category="generate_pdf", target_tool="generate_csv", is_pdf_report=true.
   - For Excel report: category="generate_pdf", target_tool="generate_excel", is_pdf_report=true.
   NOTE: Even if previous conversation turns or history generated reports or files, asking to view or export company assets in Excel, CSV, or PDF is ALWAYS "generate_pdf".
4. "delete_asset": User asks to delete or remove an asset (e.g. "hapus aset AST-001", "delete asset AST-123", "buang aset ini").
   -> category="delete_asset", target_tool="delete_asset", is_mutation=true, is_delete=true.
5. "update_asset": User asks to update, modify, edit, or change existing assets (e.g. "ubah status laptop menjadi rusak", "ganti lokasi monitor ke lantai 3").
   -> category="update_asset", target_tool="update_asset", is_mutation=true.
6. "add_asset": User asks to add, create, insert, or register new assets (e.g. "tambahkan 5 laptop Dell", "daftarkan printer baru", "masukkan data barang ini").
   -> category="add_asset", target_tool="add_asset", is_mutation=true. Strictly target_tool="add_asset", NEVER "get_assets".
7. "query_asset": User asks to view, search, find, count, or filter IT assets in the system (e.g. "tampilkan semua laptop", "berapa jumlah monitor", "cari aset di gudang").
   -> category="query_asset", target_tool="get_assets", is_off_topic=false.
8. "create_schedule": User asks to create, add, plan, or record an IT asset audit schedule, maintenance schedule, or stock-take (e.g. "buat jadwal audit", "jadwalkan audit laptop mulai 15 September secara bulanan", "tambahkan jadwal audit", "jadwal audit rutin").
   -> category="create_schedule", target_tool="create_schedule", is_mutation=true, is_off_topic=false.
9. "get_schedules": User asks to view, check, list, or inspect existing schedules or audit plans (e.g. "lihat jadwal audit", "tampilkan semua jadwal", "apa ada jadwal audit bulan ini").
   -> category="get_schedules", target_tool="get_schedules", is_mutation=false, is_off_topic=false.
10. "greeting": Greetings or questions about who you are or what capabilities you have (e.g. "halo", "hi", "apa yang bisa kamu lakukan").
   -> category="greeting", target_tool="no_tools".

IMPORTANT: Output ONLY the raw JSON object. Do not include markdown codeblocks, explanation, or conversational text.`

	var userPrompt strings.Builder
	if userContext != nil {
		hasUploaded, _ := userContext["has_user_uploaded_file"].(bool)
		if attachText, ok := userContext["attachment_text"].(string); ok && attachText != "" {
			attachName := "Attached Document"
			if name, ok := userContext["attachment_name"].(string); ok && name != "" {
				attachName = name
			}
			if hasUploaded {
				userPrompt.WriteString(fmt.Sprintf("=== USER-UPLOADED FILE FOR THIS REQUEST (%s) ===\n%s\n\n", attachName, attachText))
			} else {
				userPrompt.WriteString(fmt.Sprintf("=== PREVIOUS CONVERSATION REFERENCE DOCUMENT (%s) ===\n%s\n\n", attachName, attachText))
			}
		}
	}
	userPrompt.WriteString(fmt.Sprintf("=== USER QUERY ===\n%s", query))

	messages := []Message{
		{Role: "system", Content: systemInstruction},
		{Role: "user", Content: userPrompt.String()},
	}

	res, err := ChatGenerate(ctx, messages, nil, 4096, model, 2)
	if err != nil {
		log.Printf("[searchIntent] ChatGenerate error: %v", err)
		return &IntentResult{
			Category:   "unknown",
			TargetTool: "no_tools",
			Reason:     err.Error(),
		}
	}

	result, err := parseIntentJSON(res.RawOutput)
	if err != nil {
		log.Printf("[searchIntent] %v", err)
		return &IntentResult{
			Category:   "unknown",
			TargetTool: "no_tools",
			Reason:     err.Error(),
		}
	}

	// Deterministic Guardrail: If user asks for audit scheduling/planning, it is NEVER off-topic
	if isAuditScheduleQuery(query) {
		result.IsOffTopic = false
		qLower := strings.ToLower(query)
		if strings.Contains(qLower, "lihat") || strings.Contains(qLower, "tampilkan") || strings.Contains(qLower, "daftar") || strings.Contains(qLower, "cek") {
			if result.Category == "unrelated" || result.Category == "unknown" || result.Category == "query_asset" {
				result.Category = "get_schedules"
				result.TargetTool = "get_schedules"
				result.IsMutation = false
				result.Reason = "Asset audit schedule query is an integral part of IT Asset Management"
			}
		} else {
			if result.Category == "unrelated" || result.Category == "unknown" || result.Category == "query_asset" {
				result.Category = "create_schedule"
				result.TargetTool = "create_schedule"
				result.IsMutation = true
				result.Reason = "Asset audit scheduling is an integral part of IT Asset Management"
			}
		}
	}

	log.Printf("[searchIntent] Query: %q -> Category: %q, TargetTool: %q, IsMutation: %t, IsDelete: %t, IsPDFReport: %t, IsOffTopic: %t",
		query, result.Category, result.TargetTool, result.IsMutation, result.IsDelete, result.IsPDFReport, result.IsOffTopic)

	return result
}

// isAuditScheduleQuery checks if the query asks to create, plan, or organize an audit schedule for assets.
func isAuditScheduleQuery(query string) bool {
	q := strings.ToLower(query)
	hasAudit := strings.Contains(q, "audit") || strings.Contains(q, "stock take") || strings.Contains(q, "stock opname") || strings.Contains(q, "inventarisasi")
	if !hasAudit {
		return false
	}
	hasScheduleOrAction := strings.Contains(q, "jadwal") || strings.Contains(q, "schedule") || strings.Contains(q, "buat") ||
		strings.Contains(q, "rencana") || strings.Contains(q, "plan") || strings.Contains(q, "kapan") ||
		strings.Contains(q, "agenda") || strings.Contains(q, "susun") || strings.Contains(q, "atur") ||
		strings.Contains(q, "rutin") || strings.Contains(q, "berkala") || strings.Contains(q, "aset") || strings.Contains(q, "asset")
	return hasScheduleOrAction
}

// SearchIntent exports searchIntent for external callers and tests.
func SearchIntent(ctx context.Context, query string, userContext map[string]any, model string) *IntentResult {
	return searchIntent(ctx, query, userContext, model)
}

func hasTool(tools []mcp.Tool, name string) bool {
	cleanTarget := strings.Trim(strings.ToLower(name), "_")
	for _, t := range tools {
		if strings.EqualFold(strings.Trim(t.Name, "_"), cleanTarget) {
			return true
		}
	}
	return false
}

// CallTools orchestrates multi-turn tool calling, reasoning loops, and response generation.
func CallTools(ctx context.Context, client *client.Client, mcpTools []mcp.Tool, query string, role string, userContext map[string]any, maxIterations int, model string) (*AgentResult, error) {
	// 1. RBAC at the top: immediately enforce role permissions and filter tools
	role = strings.ToLower(strings.TrimSpace(role))
	availableTools := filterRoles(mcpTools, role)
	if len(availableTools) == 0 {
		return &AgentResult{Answer: "I do not have permissions or tools to access that information."}, nil
	}

	if model == "" {
		model = GeneratorModel
	}
	if maxIterations < 3 {
		maxIterations = 3
	}

	// 2. Resolve intent via agent function searchIntent()
	intent := searchIntent(ctx, query, userContext, model)

	// Immediate rejection: off-topic requests or prohibited file conversion
	if intent.IsOffTopic || intent.IsFileConvert || intent.Category == "unrelated" || intent.Category == "file_conversion" {
		log.Printf("[callTools] Blocked off-topic or file conversion request (%s): %q", intent.Category, query)
		return &AgentResult{
			Answer: "I do not have information or unable to do that.",
		}, nil
	}

	// 3. RBAC permission check: if intent requires a tool, verify it exists in availableTools (no hardcoding)
	if intent.TargetTool != "" && intent.TargetTool != "no_tools" && !hasTool(availableTools, intent.TargetTool) {
		log.Printf("[callTools] Blocked query (%s): tool %q not permitted for role %s", intent.Category, intent.TargetTool, role)
		if intent.IsDelete {
			return &AgentResult{Answer: "You do not have permission to delete assets."}, nil
		}
		action := strings.ReplaceAll(intent.Category, "_", " ")
		return &AgentResult{Answer: fmt.Sprintf("You do not have permission to %s.", action)}, nil
	}

	// Apply Caveman tool catalog compression skill
	availableTools = ShrinkToolCatalog(availableTools)

	resolverMap := inferResolverMap(availableTools)
	companyName := ""
	if userContext != nil {
		if comp, ok := userContext["company"].(string); ok && comp != "" {
			companyName = comp
		} else if compInfo, ok := userContext["company_info"].(map[string]any); ok {
			if compName, ok := compInfo["company_name"].(string); ok && compName != "" {
				companyName = compName
			}
		}
	}

	executedSignatures := make(map[string]bool)
	var accumulatedResults []ToolExecutionResult
	extractedFactsSoFar := ""
	missingInfoHint := ""

	toolGuide := buildToolGuide(availableTools)
	systemInstruction := fmt.Sprintf(
		"You are an internal IT Asset Management intelligent assistant and tool router for QTERA.\n"+
			"Your task is to analyze the user's question, determine necessary asset operations, and select the optimal tool(s) from your available tools.\n\n"+
			"STRICT DOMAIN BOUNDARY & OFF-TOPIC REJECTION:\n"+
			"- You are STRICTLY and EXCLUSIVELY limited to IT Asset Management (managing, tracking, querying hardware/devices/equipment, asset audit scheduling/stock-take, and generating asset PDF, CSV, and Excel reports from database inventory).\n"+
			"- ASSET AUDIT SCHEDULING & TASKS (NOT OFF-TOPIC):\n"+
			"  * Requests to create, record, or plan an audit schedule or stock-take (e.g., 'buat jadwal audit', 'jadwalkan audit laptop mulai 15 September secara bulanan', 'jadwal audit aset', 'stock opname') are legitimate IT Asset Management requests and MUST NOT be rejected as off-topic.\n"+
			"  * When the user asks to create or schedule an audit, call `create_schedule` directly with arguments: `category` (string, max 125 chars, e.g. 'Audit Aset Laptop'), `start` (string, max 125 chars, e.g. '2026-09-15' or '-' if unspecified), and `frequency` (string, max 125 chars, e.g. 'Monthly', 'Bulanan', 'Weekly', or '-' if unspecified). Fill any missing fields with \"-\".\n"+
			"  * When the user asks to view or check existing audit schedules, call `get_schedules`.\n"+
			"- You MUST NOT answer questions, write code, create poems/stories, provide recipes, or engage in conversation regarding ANY topics outside IT Asset Management.\n"+
			"- FILE CONVERSION RESTRICTION: You CANNOT and MUST NOT convert user files, spreadsheets, documents, or attachments to CSV, PDF, Excel (.xlsx), or ANY other format. File conversion is strictly prohibited.\n"+
			"- If the user asks to convert, transform, or export an uploaded file or attachment into CSV, PDF, or Excel, you MUST call the `no_tools` tool with reason \"file_conversion_prohibited\". NEVER perform file conversion or call generate_pdf, generate_csv, or generate_excel on user-uploaded files.\n\n"+
			"TOOL CAPABILITY & ACCESS BOUNDARY:\n"+
			"- You may ONLY call tools listed in the Available Tools Guide below. If a tool is not listed, you DO NOT have permission or capability to use it.\n"+
			"- If the user requests an action requiring a tool not in your Available Tools Guide (such as adding, modifying, or deleting assets), you MUST call `no_tools` with reason \"unauthorized\" and explain that you do not have permission for that action.\n"+
			"- Never invent, guess, or hallucinate tool names.\n\n"+
			"- If there is missing information in add_asset, fill it with \"-\".\n"+
			"- FOR ADDING ASSETS (`add_asset`): NEVER call `get_assets`. Call `add_asset` directly on the very first turn. You do NOT need to check or query existing assets before adding a new asset.\n"+
			"- For add_asset: Even if there is data similar or identical to existing data or one previously inputted, you MUST still input it and treat each as a separate, distinct asset entry. Never skip, merge, or ignore adding an asset because of similarity.\n"+
			"- MULTIPLE SIMILAR USER INPUTS: If the user inputs or specifies multiple similar or identical items (e.g. 'tambahkan 3 laptop Dell', or lists multiple identical items in chat/document), you MUST treat each item as a distinct, separate asset and emit one separate `add_asset` tool call for every single item. Never combine, collapse, or deduplicate them.\n"+
			"If you see an id don't change any of its format (example: AST-0001 keep and pass as AST-0001).\n"+
			"Provide the tool call directly without greeting back or unnecessary pleasantries.\n"+
			"INPUT VALIDATION MANDATE: For any input fields, arguments, or parameters that are empty or not specified (such as category, brand, modelType, location, purchasePrice, purchaseDate), you MUST fill them with \"-\". Never omit them or leave them empty.\n"+
			"INVENTORY QUERYING & LIVE DATABASE MANDATE:\n"+
			"- LIVE DATABASE PRIORITY (FOR QUERIES ONLY): When the user inquires about existing assets, equipment, devices, inventory counts, or lists (e.g. 'show me all assets in this system', 'tampilkan semua aset', 'how many laptops do we have'), you MUST call `get_assets` to fetch fresh live data over cached context. DO NOT call `get_assets` when adding new assets.\n"+
			"- NEVER assume or claim an attached document/spreadsheet or chat history contains all assets in the system. Attached documents are ONLY reference materials or data to be imported, NOT the live system inventory.\n\n"+
			"Available Tools Guide:\n%s\n\n"+
			"Execution Rules:\n"+
			"1. **Thinking (<thought>...</thought>)**: Identify the user's intent. If adding assets, choose `add_asset` immediately without calling `get_assets`.\n"+
			"2. **Tool Execution**: Call only tools from the Available Tools Guide. For asset additions, invoke `add_asset` directly without prior lookup. When inserting multiple items from user inputs or documents (e.g. 5 laptops or repeated similar entries), emit one individual `add_asset` call per item. Even if items have similar or identical specifications or names, treat each as different data and input every single one. Fill any missing or empty input arguments with \"-\".\n"+
			"3. **Updating Assets by Name (Multi-Step Lookup)**: To call `update_asset`, the unique `id` (e.g. 'AST-XXXXXXXX') is REQUIRED. If the user asks to update or modify an asset by name or keyword without providing the exact ID (e.g. 'update termostat price to 150000'), you MUST FIRST call `get_assets` with `search` set to the asset name (e.g. `{\"name\": \"get_assets\", \"arguments\": {\"search\": \"termostat\"}}`) to look up its unique ID. Do NOT call `no_tools`! Once the ID is retrieved, call `update_asset` with that ID.\n"+
			"4. **Company Scoping**: Asset operations are strictly validated and automatically scoped to the user's company.\n"+
			"5. **Audit Scheduling**: When the user requests creating, adding, or planning a schedule for asset audit or stock-take, call `create_schedule` directly. If the user asks to view or check schedules, call `get_schedules`.\n"+
			"6. **No Tools Needed**: For greetings, unauthorized actions, or off-topic prompts, call `no_tools`.\n\n"+
			"Output format:\n"+
			"If authorized tools are needed:\n"+
			"<thought>Brief reasoning on asset data needed</thought>\n"+
			"<tool_call>\n{\"name\": \"tool_name\", \"arguments\": {...}}\n</tool_call>\n\n"+
			"If NO tools are needed (greeting, unauthorized action, or off-topic prompt):\n"+
			"<thought>Brief reasoning</thought>\n"+
			"<tool_call>\n{\"name\": \"no_tools\", \"arguments\": {\"reason\": \"greeting, unauthorized, or unrelated_topic\"}}\n</tool_call>",
		toolGuide,
	)

	if companyName != "" {
		systemInstruction += fmt.Sprintf("\nUser belongs to company '%s'.", companyName)
	}

	var history []ChatMessage
	if userContext != nil {
		if h, ok := userContext["history"].([]ChatMessage); ok {
			history = h
		}
	}
	historyStr := FormatChatHistoryForLlm(history)

	var routerUserPrompt strings.Builder
	if historyStr != "" {
		routerUserPrompt.WriteString(fmt.Sprintf("=== RECENT CONVERSATION HISTORY ===\n%s\n\n", historyStr))
	}
	if userContext != nil {
		if attachText, ok := userContext["attachment_text"].(string); ok && attachText != "" {
			attachName := "Attached Document"
			if name, ok := userContext["attachment_name"].(string); ok && name != "" {
				attachName = name
			}
			routerUserPrompt.WriteString(fmt.Sprintf("=== ATTACHED DOCUMENT CONTEXT (%s) ===\n%s\n\n", attachName, attachText))
		}
	}
	routerUserPrompt.WriteString(fmt.Sprintf("=== QUESTION ===\n%s", query))

	// 3. Conversational message history maintained across iterations
	messages := []Message{
		{Role: "system", Content: systemInstruction},
		{Role: "user", Content: routerUserPrompt.String()},
	}

	for iteration := 1; iteration <= maxIterations; iteration++ {
		res, err := ChatGenerate(ctx, messages, nil, 32768, model, 3)
		if err != nil {
			log.Printf("[callTools] ChatGenerate error on iteration %d: %v", iteration, err)
			break
		}
		log.Printf("[callTools] Iteration %d tokens -> input: %d, output: %d, total: %d", iteration, res.InputTokens, res.OutputTokens, res.InputTokens+res.OutputTokens)

		rawOutput := res.RawOutput
		log.Printf("[callTools] Iteration %d LLM output: %s", iteration, rawOutput)

		toolCalls := tryParseToolCalls(rawOutput)

		// Filter out any tool calls not present in availableTools
		var authorizedCalls []ToolCall
		for _, tc := range toolCalls {
			if tc.Name == "no_tools" || tc.Name == "no_tool" || hasTool(availableTools, tc.Name) {
				authorizedCalls = append(authorizedCalls, tc)
			} else {
				log.Printf("[callTools] Dropped unauthorized tool call %q (not available in session)", tc.Name)
			}
		}

		toolCalls = resolveIdentifiers(authorizedCalls, availableTools, accumulatedResults, resolverMap, query)

		var newCalls []ToolCall
		for _, tc := range toolCalls {
			argsJson, _ := json.Marshal(tc.Arguments)
			sig := fmt.Sprintf("%s:%s", tc.Name, string(argsJson))

			if tc.Name == "add_asset" {
				// For asset insertion: allow identical calls within the same batch (e.g. 5 identical chairs in a spreadsheet)
				// but prevent repeating already-executed calls across subsequent iterations
				if iteration > 1 && executedSignatures[sig] {
					continue
				}
				newCalls = append(newCalls, tc)
			} else {
				// For get_assets, update_asset, delete_asset: deduplicate redundant calls
				if !executedSignatures[sig] {
					executedSignatures[sig] = true
					newCalls = append(newCalls, tc)
				}
			}
		}

		for _, tc := range newCalls {
			if tc.Name == "add_asset" {
				argsJson, _ := json.Marshal(tc.Arguments)
				sig := fmt.Sprintf("%s:%s", tc.Name, string(argsJson))
				executedSignatures[sig] = true
			}
		}

		hasRealTools := false
		var realCalls []ToolCall
		for _, tc := range newCalls {
			if tc.Name != "no_tools" && tc.Name != "no_tool" {
				hasRealTools = true
				realCalls = append(realCalls, tc)
			}
		}

		if !hasRealTools {
			// If the user's intent requires a tool available in this session, guide the model to execute it
			if iteration < maxIterations && hasTool(availableTools, intent.TargetTool) {
				promptText := fmt.Sprintf("Please proceed with the operation by calling the appropriate tool (%s) from your Available Tools Guide. For any missing or unspecified input fields, fill them with \"-\".", intent.TargetTool)
				messages = append(messages,
					Message{Role: "assistant", Content: rawOutput},
					Message{Role: "user", Content: promptText},
				)
				continue
			}
			cleanReply := SanitizeModelReply(rawOutput)
			if cleanReply != "" && cleanReply == "I do not have information or unable to do that." && (intent.IsOffTopic || intent.IsFileConvert || intent.Category == "unrelated") {
				return &AgentResult{Answer: cleanReply}, nil
			}
			if iteration == 1 {
				// Filter history and context with extractContext first, then generate governed response
				_, facts, _ := extractContext(ctx, query, intent, accumulatedResults, userContext, iteration, maxIterations, model)
				return &AgentResult{Answer: GenerateResponse(ctx, query, facts, userContext, model)}, nil
			}
			if len(accumulatedResults) > 0 {
				_, facts, _ := extractContext(ctx, query, intent, accumulatedResults, userContext, iteration, maxIterations, model)
				attach := extractGeneratedAttachment(accumulatedResults)
				return &AgentResult{
					Answer:     GenerateResponse(ctx, query, facts, userContext, model),
					Attachment: attach,
				}, nil
			}
			if cleanReply != "" {
				return &AgentResult{Answer: cleanReply}, nil
			}
			return &AgentResult{Answer: "I do not have information or unable to do that."}, nil
		}

		newCalls = realCalls

		// Execution barrier: Prevent calling generate_pdf, generate_csv, or generate_excel on user files/attachments without live database queries
		hasUserUploadedFile := false
		if userContext != nil {
			if b, ok := userContext["has_user_uploaded_file"].(bool); ok {
				hasUserUploadedFile = b
			}
		}

		hasQueriedAssets := false
		for _, ar := range accumulatedResults {
			if ar.Tool == "get_assets" {
				hasQueriedAssets = true
				break
			}
		}

		for _, tc := range newCalls {
			if tc.Name == "generate_pdf" || tc.Name == "generate_csv" || tc.Name == "generate_excel" {
				if hasUserUploadedFile && !hasQueriedAssets {
					log.Printf("[callTools] Blocked prohibited file conversion attempt: tool %s called on user-uploaded file without live database query", tc.Name)
					return &AgentResult{
						Answer: "I do not have information or unable to do that.",
					}, nil
				}
			}
		}

		execResults := executeToolsParallel(ctx, client, newCalls, availableTools, userContext)
		accumulatedResults = append(accumulatedResults, execResults...)

		isEnough, facts, missing := extractContext(ctx, query, intent, accumulatedResults, userContext, iteration, maxIterations, model)
		extractedFactsSoFar = facts
		missingInfoHint = missing

		if intent.IsPDFReport && extractGeneratedAttachment(accumulatedResults) == nil && iteration < maxIterations {
			isEnough = false
			target := intent.TargetTool
			if target == "" {
				target = "generate_pdf"
			}
			missingInfoHint = fmt.Sprintf("Asset data has been retrieved. You MUST now call the '%s' tool with the relevant assets formatted into 'headers', 'data', and 'title' so the document can be created.", target)
		}

		if isEnough || iteration >= maxIterations {
			break
		}

		// Keep true conversational history in the loop for the next iteration:
		messages = append(messages, Message{Role: "assistant", Content: rawOutput})
		toolResultText := formatMultiRawDataForLlm(execResults)
		userFeedback := fmt.Sprintf("Tool Execution Results:\n%s", toolResultText)
		if missingInfoHint != "" {
			userFeedback += fmt.Sprintf("\nMissing information needed to complete the user's request: %s\nProceed with the next tool call.", missingInfoHint)
		}
		messages = append(messages, Message{Role: "user", Content: userFeedback})
	}

	generatedAttach := extractGeneratedAttachment(accumulatedResults)
	if generatedAttach != nil {
		extractedFactsSoFar += fmt.Sprintf("\n\n=== GENERATED ATTACHMENT ===\nA file report has been generated successfully and attached to this message:\n- Filename: %s\n- URL: %s\nInform the user that the report has been generated and is attached to this message for them to download directly.", generatedAttach.Name, generatedAttach.URL)
	}

	return &AgentResult{
		Answer:     GenerateResponse(ctx, query, extractedFactsSoFar, userContext, model),
		Attachment: generatedAttach,
	}, nil
}

func extractGeneratedAttachment(accumulated []ToolExecutionResult) *AttachmentInfo {
	for i := len(accumulated) - 1; i >= 0; i-- {
		res := accumulated[i]
		toolLower := strings.ToLower(res.Tool)
		if strings.Contains(toolLower, "generate_pdf") || strings.Contains(toolLower, "pdf") ||
			strings.Contains(toolLower, "generate_csv") || strings.Contains(toolLower, "csv") ||
			strings.Contains(toolLower, "generate_excel") || strings.Contains(toolLower, "excel") {
			if m, ok := res.Result.(map[string]any); ok {
				downloadURL, _ := m["download_url"].(string)
				filename, _ := m["filename"].(string)
				if downloadURL != "" {
					if filename == "" {
						filename = filepath.Base(downloadURL)
					}
					var size int64
					if sNum, ok := m["size"].(float64); ok {
						size = int64(sNum)
					} else if sInt, ok := m["size"].(int64); ok {
						size = sInt
					} else if sInt, ok := m["size"].(int); ok {
						size = int64(sInt)
					}
					mimeType, _ := m["type"].(string)
					if mimeType == "" {
						ext := strings.ToLower(filepath.Ext(filename))
						switch ext {
						case ".csv":
							mimeType = "text/csv"
						case ".xlsx":
							mimeType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
						default:
							mimeType = "application/pdf"
						}
					}
					return &AttachmentInfo{
						Name: filename,
						URL:  downloadURL,
						Type: mimeType,
						Size: size,
					}
				}
			}
		}
	}
	return nil
}

func cleanContextPayload(contextStr string) string {
	trimmed := strings.TrimSpace(contextStr)
	if trimmed == "" {
		return "No records found."
	}
	reConsecutiveNewlines := regexp.MustCompile(`\n{3,}`)
	return reConsecutiveNewlines.ReplaceAllString(trimmed, "\n\n")
}

// GenerateResponse generates the final response
func GenerateResponse(ctx context.Context, query string, contextStr string, userContext map[string]any, model string) string {
	if model == "" {
		model = GeneratorModel
	}

	systemPrompt := "You are QTERA AI, a specialized IT Asset Management assistant.\n" +
		"Your scope is STRICTLY and EXCLUSIVELY limited to IT Asset Management (managing, tracking, querying, mutating IT hardware, equipment, devices, inventory, and generating asset PDF reports from database inventory).\n\n" +
		"Strict Guidelines:\n" +
		"1. LIVE DATABASE PRIORITY: Live data fetched from API tools is the single authoritative source of truth for assets in the system. Always prioritize presenting the live database records over any cached documents, attachments, or past conversation.\n" +
		"2. Attached Documents vs Live Inventory: Attached documents or spreadsheets are ONLY reference materials or data to be imported. NEVER state or imply that an attached document (e.g. data_barang.xlsx) represents all assets in the system when answering system inventory inquiries. If the user asks about assets in the system (e.g. 'show me all assets in this system', 'list all assets'), ALWAYS present the live database records.\n" +
		"3. Factual Accuracy: If a detail isn't in the context, do not mention it, invent it, or assume it.\n" +
		"4. Tone: Answer in plain, friendly, concise sentences.\n" +
		"5. PDF Reports: If a generated PDF attachment is noted in the context, explicitly confirm to the user that the PDF document has been generated and attached to this message so they can download it directly below. Do NOT claim that you cannot generate or send PDF files.\n" +
		"6. No Matching Records: If the context indicates no matching records were found for a specific filter/time period, state clearly that no matching items were found (e.g. 'No assets were found for this category.').\n" +
		"7. Greetings, Identity & Asset Audit Schedules: If the user sends a standard greeting (e.g., 'Halo', 'Hi', 'Hello', 'Selamat pagi') or asks about what you can do or how to use the system, respond warmly and briefly, identifying yourself as QTERA AI and stating you assist with IT Asset Management. If the user asks to create, suggest, or organize an asset audit schedule (e.g. 'buat jadwal audit', 'jadwal audit aset', 'jadwalkan audit laptop kantor'), this is fully within scope. Provide a structured, helpful, and professional audit plan or timeline based on the assets in the context.\n" +
		"8. STRICT OFF-TOPIC REJECTION: You MUST NOT answer questions, give instructions, generate code, write creative content, convert user files/attachments to PDF, or converse about ANY topics unrelated to IT Asset Management (e.g., general knowledge, programming/coding, cooking/recipes, science, math, translation, trivia, news, politics, entertainment, personal advice, or general casual chatter).\n" +
		"   If the user's question is unrelated to IT Asset Management, or asks to convert a file to PDF, or if the information is completely missing or unable to answer, you MUST respond EXACTLY with:\n" +
		"   'I do not have information or unable to do that.'\n" +
		"   Do NOT attempt to fulfill unrelated requests, do NOT apologize, and do NOT explain why.\n" +
		"9. Missing Data Values: When presenting or describing asset records with missing or empty values, display them as '-'.\n"

	cleanedContext := cleanContextPayload(contextStr)

	var userPromptBuilder strings.Builder
	if userContext != nil {
		if attachText, ok := userContext["attachment_text"].(string); ok && attachText != "" {
			attachName := "Attached Document"
			if name, ok := userContext["attachment_name"].(string); ok && name != "" {
				attachName = name
			}
			userPromptBuilder.WriteString(fmt.Sprintf("=== ATTACHED DOCUMENT CONTEXT (%s) ===\n%s\n\n", attachName, attachText))
		}
	}
	userPromptBuilder.WriteString(fmt.Sprintf("=== CONTEXT DATA (FILTERED & EXTRACTED) ===\n%s\n\n=== QUESTION ===\n%s", cleanedContext, query))

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPromptBuilder.String()},
	}

	res, err := ChatGenerate(ctx, messages, nil, 32768, model, 3)
	if err != nil {
		log.Printf("[generateResponse] Failed to generate response: %v", err)
		return "I do not have information or unable to do that."
	}
	log.Printf("[generateResponse] tokens -> input: %d, output: %d, total: %d", res.InputTokens, res.OutputTokens, res.InputTokens+res.OutputTokens)
	finalAns := SanitizeModelReply(res.RawOutput)
	if finalAns == "" {
		return "I do not have information or unable to do that."
	}
	return finalAns
}
