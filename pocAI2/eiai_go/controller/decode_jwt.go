package controller

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func JWTDecode(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	var payload map[string]any
	if len(parts) >= 2 {
		payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err == nil {
			_ = json.Unmarshal(payloadBytes, &payload)
		}
	}
	return payload, nil
}

func FetchUserContext(token string, defaultRole string) map[string]any {
	cleanRole := strings.ToLower(strings.TrimSpace(defaultRole))
	if cleanRole == "" {
		cleanRole = "operator"
	}
	cleanToken := ""
	if token != "" {
		cleanToken = strings.TrimSpace(strings.ReplaceAll(token, "Bearer ", ""))
	}

	if cleanToken == "" {
		return map[string]any{
			"user_id":  "anonymous",
			"username": "Anonymous",
			"role":     cleanRole,
			"email":    "",
			"company":  "",
			"token":    "",
		}
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:3000/api"
	}

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", baseURL+"/auth/me", nil)
	if err == nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cleanToken))
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				data := map[string]any{}
				_ = json.NewDecoder(resp.Body).Decode(&data)
				u, ok := data["user"].(map[string]any)
				if !ok {
					u = map[string]any{}
				}

				role, _ := u["role"].(string)
				if role != "" {
					role = strings.ToLower(strings.TrimSpace(role))
				} else {
					role = cleanRole
				}

				return map[string]any{
					"user_id":  fmt.Sprintf("%v", u["id"]),
					"username": fmt.Sprintf("%v", u["username"]),
					"email":    fmt.Sprintf("%v", u["email"]),
					"role":     role,
					"company":  fmt.Sprintf("%v", u["company"]),
					"token":    cleanToken,
				}
			}
		}
	}

	// Fallback: decode claims directly from JWT token
	payload, _ := JWTDecode(cleanToken)
	if payload != nil {
		role1, _ := payload["role"].(string)
		if role1 != "" {
			role1 = strings.ToLower(strings.TrimSpace(role1))
		} else {
			role1 = cleanRole
		}

		return map[string]any{
			"user_id":  fmt.Sprintf("%v", payload["id"]),
			"username": fmt.Sprintf("%v", payload["username"]),
			"email":    fmt.Sprintf("%v", payload["email"]),
			"role":     role1,
			"company":  fmt.Sprintf("%v", payload["company"]),
			"token":    cleanToken,
		}
	}

	return map[string]any{
		"user_id":  "anonymous",
		"username": "Anonymous",
		"role":     cleanRole,
		"email":    "",
		"company":  "",
		"token":    cleanToken,
	}
}

type AttachmentInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

type ChatMessage struct {
	ID         string          `json:"id,omitempty"`
	ChatID     string          `json:"chat_id,omitempty"`
	UserID     string          `json:"user_id,omitempty"`
	Username   string          `json:"username,omitempty"`
	Role       string          `json:"role,omitempty"`
	User       bool            `json:"user"`
	Chat       string          `json:"chat"`
	Models     string          `json:"models,omitempty"`
	CreatedAt  string          `json:"created_at,omitempty"`
	Attachment *AttachmentInfo `json:"attachment,omitempty"`
}

var (
	attachmentCacheMu sync.RWMutex
	attachmentCache   = make(map[string]string)
)

func GetCachedAttachmentText(key string) string {
	attachmentCacheMu.RLock()
	defer attachmentCacheMu.RUnlock()
	return attachmentCache[key]
}

func SetCachedAttachmentText(key, text string) {
	attachmentCacheMu.Lock()
	defer attachmentCacheMu.Unlock()
	attachmentCache[key] = text
}

func FetchAttachmentContent(attachURL string, token string) ([]byte, error) {
	cleanPath := strings.TrimPrefix(attachURL, "/")
	localCandidates := []string{
		filepath.Join("..", "server", "GO", filepath.FromSlash(cleanPath)),
		filepath.Join("server", "GO", filepath.FromSlash(cleanPath)),
		filepath.FromSlash(cleanPath),
	}
	for _, cand := range localCandidates {
		if data, err := os.ReadFile(cand); err == nil && len(data) > 0 {
			return data, nil
		}
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:3000"
	}
	cleanBase := strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/api")
	targetURL := cleanBase + "/" + cleanPath

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		cleanToken := strings.TrimSpace(strings.ReplaceAll(token, "Bearer ", ""))
		req.Header.Set("Authorization", "Bearer "+cleanToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch attachment: status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// FetchChatHistory retrieves the last N messages for a given chatID from the server.
func FetchChatHistory(chatID string, token string, limit int) []ChatMessage {
	if chatID == "" {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:3000"
	}
	cleanBase := strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/api")
	messagesURL := cleanBase + "/messages/" + chatID

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", messagesURL, nil)
	if err != nil {
		log.Printf("[FetchChatHistory] NewRequest error: %v", err)
		return nil
	}

	cleanToken := strings.TrimSpace(strings.ReplaceAll(token, "Bearer ", ""))
	if cleanToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cleanToken))
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[FetchChatHistory] HTTP GET error: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[FetchChatHistory] Failed to get messages from %s (status %d)", messagesURL, resp.StatusCode)
		return nil
	}

	var allMessages []ChatMessage
	if err := json.NewDecoder(resp.Body).Decode(&allMessages); err != nil {
		log.Printf("[FetchChatHistory] JSON decode error: %v", err)
		return nil
	}

	if len(allMessages) > limit {
		allMessages = allMessages[len(allMessages)-limit:]
	}

	// If the last message is the current in-flight user message, trim it so history only contains prior turns
	if len(allMessages) > 0 && allMessages[len(allMessages)-1].User {
		allMessages = allMessages[:len(allMessages)-1]
	}

	log.Printf("[FetchChatHistory] Successfully fetched %d history messages for chatID '%s'", len(allMessages), chatID)
	return allMessages
}

// FormatChatHistoryForLlm formats a list of chat messages into a compact context string for the LLM.
func FormatChatHistoryForLlm(history []ChatMessage) string {
	if len(history) == 0 {
		return ""
	}

	var lines []string
	for _, m := range history {
		role := "User"
		if !m.User {
			role = "Assistant"
		}
		text := strings.TrimSpace(m.Chat)
		if text != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", role, text))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}
