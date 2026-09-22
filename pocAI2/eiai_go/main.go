package main

import (
	"context"
	"io"
	"log"
	"os"
	"strings"
	"time"

	controller "eiai_go/controller"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var (
	mcpClient *client.Client
	mcpTools  []mcp.Tool
)

func main() {
	_ = godotenv.Load(".env")

	// 1. Connect to MCP server
	c, err := controller.NewServerConnection()
	if err != nil {
		log.Fatalf("MCP connect error: %v", err)
	}
	mcpClient = c

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	toolsResp, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("MCP list tools error: %v", err)
	}
	mcpTools = toolsResp.Tools

	// 2. Start HTTP Server
	app := fiber.New()
	app.Use(cors.New())

	app.All("/rag", func(c *fiber.Ctx) error {
		query := strings.TrimSpace(c.Query("query"))
		if query == "" {
			query = strings.TrimSpace(c.FormValue("query"))
		}
		if query == "" {
			return c.Status(400).SendString("query is required")
		}

		modelsParam := c.Query("models")
		if modelsParam == "" {
			modelsParam = c.FormValue("models")
		}
		model := controller.ResolveModel(modelsParam)

		role := c.Query("role")
		if role == "" {
			role = c.FormValue("role", "operator")
		}
		token := c.Query("token")
		if token == "" {
			token = c.FormValue("token")
		}
		if token == "" {
			token = c.Get("Authorization")
		}

		// Support Header Auth (Tag Samurai BETS-V2 style: X-User-ID & X-User-Role)
		headerUserID := strings.TrimSpace(c.Get("X-User-ID"))
		if headerUserID == "" {
			headerUserID = strings.TrimSpace(c.Query("user_id"))
		}
		if headerUserID == "" {
			headerUserID = strings.TrimSpace(c.FormValue("user_id"))
		}

		headerUserRole := strings.TrimSpace(c.Get("X-User-Role"))
		if headerUserRole == "" {
			headerUserRole = strings.TrimSpace(c.Get("X-Role"))
		}
		if headerUserRole != "" {
			role = strings.ToLower(headerUserRole)
		}

		var userContext map[string]any
		if headerUserID != "" {
			headerUsername := strings.TrimSpace(c.Get("X-User-Name"))
			if headerUsername == "" {
				headerUsername = strings.TrimSpace(c.Get("X-Username"))
			}
			if headerUsername == "" {
				headerUsername = "User_" + headerUserID
			}
			headerEmail := strings.TrimSpace(c.Get("X-User-Email"))
			userContext = map[string]any{
				"user_id":  headerUserID,
				"username": headerUsername,
				"email":    headerEmail,
				"role":     role,
				"token":    token,
			}
		} else {
			userContext = controller.FetchUserContext(token, role)
		}
		if r, ok := userContext["role"].(string); ok && r != "" {
			role = r
		}
		company := strings.TrimSpace(c.Get("X-Company"))
		if company == "" {
			company = strings.TrimSpace(c.Query("company"))
		}
		if company == "" {
			company = strings.TrimSpace(c.FormValue("company"))
		}
		if company != "" {
			userContext["company"] = company
		}

		chatID := c.Query("chat_id")
		if chatID == "" {
			chatID = c.FormValue("chat_id")
		}
		if chatID == "" {
			if uid, ok := userContext["user_id"].(string); ok && uid != "" && uid != "anonymous" {
				chatID = "user_" + uid
			} else if uname, ok := userContext["username"].(string); ok && uname != "" && uname != "Anonymous" {
				chatID = "user_" + uname
			}
		}

		var history []controller.ChatMessage
		if chatID != "" {
			history = controller.FetchChatHistory(chatID, token, 10)
			if len(history) > 0 {
				userContext["history"] = history
			}
		}

		var activeAttachName string
		var activeAttachText string
		hasUserUploadedFile := false

		// 1. Handle attached file (PDF, image, text, CSV, etc.) from current request
		if fileHeader, err := c.FormFile("attachment"); err == nil && fileHeader != nil {
			hasUserUploadedFile = true
			f, err := fileHeader.Open()
			if err == nil {
				defer f.Close()
				content, err := io.ReadAll(f)
				if err == nil && len(content) > 0 {
					extractedText, extractErr := controller.ExtractText(content, fileHeader.Filename)
					if extractErr == nil && strings.TrimSpace(extractedText) != "" {
						activeAttachName = fileHeader.Filename
						activeAttachText = strings.TrimSpace(extractedText)
						controller.SetCachedAttachmentText(fileHeader.Filename, activeAttachText)
						log.Printf("[main] Extracted %d chars from attachment %q", len(extractedText), fileHeader.Filename)
					} else if extractErr != nil {
						log.Printf("[main] ExtractText error for %q: %v", fileHeader.Filename, extractErr)
					}
				}
			}
		}

		// 2. If no attachment in current request, inspect recent history for an active attachment
		if activeAttachText == "" && len(history) > 0 {
			for i := len(history) - 1; i >= 0; i-- {
				msg := history[i]
				if msg.Attachment != nil && msg.Attachment.Name != "" {
					attachKey := msg.Attachment.URL
					if attachKey == "" {
						attachKey = msg.Attachment.Name
					}
					if cached := controller.GetCachedAttachmentText(attachKey); cached != "" {
						activeAttachName = msg.Attachment.Name
						activeAttachText = cached
						log.Printf("[main] Reused cached attachment text (%d chars) from history: %q", len(cached), activeAttachName)
						break
					}
					if msg.Attachment.URL != "" {
						content, fetchErr := controller.FetchAttachmentContent(msg.Attachment.URL, token)
						if fetchErr == nil && len(content) > 0 {
							extText, extErr := controller.ExtractText(content, msg.Attachment.Name)
							if extErr == nil && strings.TrimSpace(extText) != "" {
								activeAttachName = msg.Attachment.Name
								activeAttachText = strings.TrimSpace(extText)
								controller.SetCachedAttachmentText(attachKey, activeAttachText)
								log.Printf("[main] Retrieved & extracted %d chars for history attachment: %q", len(activeAttachText), activeAttachName)
								break
							}
						}
					}
				}
			}
		}

		if activeAttachText != "" {
			userContext["attachment_name"] = activeAttachName
			userContext["attachment_text"] = activeAttachText
		}
		userContext["has_user_uploaded_file"] = hasUserUploadedFile

		res, err := controller.CallTools(c.Context(), mcpClient, mcpTools, query, role, userContext, 2, model)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		return c.JSON(res)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Printf("[main] Starting EIAI Go server on :%s ...", port)
	log.Fatal(app.Listen(":" + port))
}
