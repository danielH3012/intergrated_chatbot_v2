package controller

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp_server/client"
)

// RegisterUtilityTools mounts utility and fallback tools onto the MCP server.
func RegisterUtilityTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("no_tools",
		mcp.WithDescription(
			`Call this tool when the task or question does NOT require any tools (e.g. greetings, casual conversation, general questions, or clarification that does not require querying or modifying the asset database).
This tool performs no action and signals that no external tools are needed for this turn.
Parameters:
  - reason: Optional explanation of why no tools are needed (e.g. 'General greeting', 'Casual conversation', 'Non-asset question')`),
		mcp.WithString("reason", mcp.Description("Optional explanation why no tools are needed")),
	), HandleNoTools)
}

// HandleNoTools executes when no external tools or database operations are needed.
func HandleNoTools(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	reason := client.GetArgString(req, "reason", "message", "explanation")
	if client.Logger != nil {
		client.Logger.Printf("[HandleNoTools] Executing no_tools | reason=%s", reason)
	}
	return client.ToolResult(map[string]any{
		"ok":      true,
		"status":  "no_tools_needed",
		"message": "No external tools or database operations required for this query.",
		"reason":  reason,
	})
}
