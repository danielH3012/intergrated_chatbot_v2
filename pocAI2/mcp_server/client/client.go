package client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// Global configuration & logger
var (
	BaseURL = GetEnv("BASE_URL", "http://127.0.0.1:3000/api")
	ApiKey  = GetEnv("API_KEY", "")
	Logger  *log.Logger
)

func GetEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// InitLogger initializes file-based logging for the MCP server.
func InitLogger() {
	f, err := os.OpenFile("mcp_server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	Logger = log.New(f, "", log.LstdFlags|log.Lshortfile)
	Logger.Println("mcp server started")
	if ApiKey == "" {
		Logger.Println("WARNING: API_KEY is empty — requests to BASE_URL will likely fail auth if protected")
	}
}

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

func BuildHeaders(creds *Credentials) http.Header {
	h := http.Header{}
	if ApiKey != "" {
		h.Set("Authorization", "Bearer "+ApiKey)
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
			h.Set("X-User-Role", creds.Role)
		}
	}
	return h
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// DoRequest makes an authenticated HTTP request to the core backend API.
func DoRequest(method, path string, params url.Values, body any, creds *Credentials) (any, error) {
	u := strings.TrimRight(BaseURL, "/") + path
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
	req.Header = BuildHeaders(creds)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if Logger != nil {
			Logger.Printf("%s %s failed: %v", method, u, err)
		}
		return map[string]string{"error": err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("%s %s returned status %d", method, u, resp.StatusCode)
		if Logger != nil {
			Logger.Println(errMsg)
		}
		return map[string]string{"error": errMsg}, nil
	}

	if resp.StatusCode == http.StatusNoContent {
		return map[string]any{"ok": true, "status": "deleted"}, nil
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		if Logger != nil {
			Logger.Printf("%s %s read failed: %v", method, u, err)
		}
		return map[string]string{"error": err.Error()}, nil
	}

	if len(respBytes) == 0 {
		return map[string]any{"ok": true}, nil
	}

	var parsed any
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return string(respBytes), nil
	}
	return parsed, nil
}

// ExtractCredentials retrieves credentials object injected into CallToolRequest arguments.
func ExtractCredentials(req mcp.CallToolRequest) *Credentials {
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

// GetArgString safely retrieves a string value from request arguments by checking multiple alias keys.
func GetArgString(req mcp.CallToolRequest, keys ...string) string {
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

// ToolResult wraps any serializable data into an MCP CallToolResult.
func ToolResult(data any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

// ErrResult returns a formatted error result to the LLM.
func ErrResult(msg string) (*mcp.CallToolResult, error) {
	return ToolResult(map[string]string{"error": msg})
}
