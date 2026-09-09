package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang-dh/middleware"
	"golang-dh/models"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)


type hub struct {
	mu    sync.Mutex
	conns map[string]map[chan []byte]struct{}
}

var wsHub = &hub{conns: make(map[string]map[chan []byte]struct{})}

func (h *hub) Register(chatID string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[chatID] == nil {
		h.conns[chatID] = make(map[chan []byte]struct{})
	}
	h.conns[chatID][ch] = struct{}{}
}

func (h *hub) Unregister(chatID string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns := h.conns[chatID]; conns != nil {
		delete(conns, ch)
		close(ch)
	}
}

func (h *hub) Broadcast(chatID string, msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.conns[chatID] {
		select {
		case ch <- msg:
		default:
		}
	}
}

func persistMessage(msgID, text, model, userID, username, role string, isUser bool, attach *models.AttachmentInfo, attachText string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if msgID == "" {
		msgID = uuid.NewString()
	}

	createdAtStr := time.Now().Format("2006-01-02 15:04:05")

	var uid any = nil
	if parsedID, err := strconv.ParseInt(userID, 10, 64); err == nil && parsedID > 0 {
		uid = parsedID
	}

	var attachName, attachURL, attachType any = nil, nil, nil
	var attachSize any = nil
	var aText any = nil
	if attach != nil && attach.Name != "" {
		attachName = attach.Name
		attachURL = attach.URL
		attachType = attach.Type
		attachSize = attach.Size
	}
	if attachText != "" {
		aText = attachText
	}

	_, err := DB.Exec(ctx,
		`INSERT INTO chat (id, "user", chat, created_at, user_id, username, role, models,
		                   attachment_name, attachment_url, attachment_type, attachment_size, attachment_text)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		msgID, isUser, text, createdAtStr, uid, username, role, model,
		attachName, attachURL, attachType, attachSize, aText,
	)
	if err != nil {
		fmt.Printf("[persistMessage] Error: %v\n", err)
	}
}


func SendMessages(c *websocket.Conn) {
	rawChatID := c.Params("id")
	chatID := strings.TrimPrefix(rawChatID, "user_")
	send := make(chan []byte, 16)
	done := make(chan struct{})

	wsHub.Register(chatID, send)
	// Also register rawChatID in case frontend connects with user_ prefix
	if rawChatID != chatID {
		wsHub.Register(rawChatID, send)
	}
	defer func() {
		wsHub.Unregister(chatID, send)
		if rawChatID != chatID {
			wsHub.Unregister(rawChatID, send)
		}
	}()

	go func() {
		for {
			select {
			case msg := <-send:
				if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			close(done)
			break
		}
		wsHub.Broadcast(chatID, msg)
		if rawChatID != chatID {
			wsHub.Broadcast(rawChatID, msg)
		}
	}
}

func SendMessage(c *fiber.Ctx) error {
	var chatID, text, models string
	var attachment *multipart.FileHeader

	if strings.Contains(c.Get("Content-Type"), "multipart/form-data") {
		chatID = c.FormValue("chat_id")
		text = c.FormValue("text")
		models = c.FormValue("models")
		if fh, err := c.FormFile("attachment"); err == nil {
			attachment = fh
		}
	} else {
		var body struct {
			ChatID string `json:"chat_id"`
			Answer string `json:"text"`
			Models string `json:"models"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		chatID = body.ChatID
		text = body.Answer
		models = body.Models
	}

	userID := "0"
	username := "Anonymous"
	role := "operator"

	if userVal := c.Locals("user"); userVal != nil {
		if claims, ok := userVal.(*middleware.JWTClaims); ok && claims != nil {
			userID = claims.UserID
			username = claims.Username
			role = claims.Role
		}
	}

	if chatID != "" {
		cleanChatID := strings.TrimPrefix(chatID, "user_")
		if cleanChatID != "" {
			chatID = cleanChatID
		}
	} else {
		chatID = username
	}
	if username == "Anonymous" && chatID != "Anonymous" {
		username = chatID
	}

	if models == "" {
		models = "ox-alpha-free"
	}

	return ProcessAndDispatchMessageWithAttachment(c, chatID, text, models, userID, username, role, attachment)
}

// ProcessAndDispatchMessage handles broadcasting user messages, calling EIAI RAG, and broadcasting the AI reply.
func ProcessAndDispatchMessage(c *fiber.Ctx, chatID, text, modelChoice, userID, username, role string) error {
	return ProcessAndDispatchMessageWithAttachment(c, chatID, text, modelChoice, userID, username, role, nil)
}

func ProcessAndDispatchMessageWithAttachment(c *fiber.Ctx, chatID, text, modelChoice, userID, username, role string, attachment *multipart.FileHeader) error {
	base := os.Getenv("EIAI_URL")
	if base == "" {
		base = "http://127.0.0.1:8000/rag"
	}

	// If an attachment is present, append the label to the text so it matches
	// the frontend's optimistic display and is preserved in chat history.
	if attachment != nil && !strings.Contains(text, "[Attached:") {
		text = strings.TrimSpace(fmt.Sprintf("%s\n📎 [Attached: %s]", text, attachment.Filename))
	}

	// Save attachment to disk permanently if present
	var attachInfo *models.AttachmentInfo
	if attachment != nil {
		uploadDir := filepath.Join("./uploads", "attachments", chatID)
		_ = os.MkdirAll(uploadDir, 0755)

		safeFilename := filepath.Base(attachment.Filename)
		diskFilename := fmt.Sprintf("%s_%s", uuid.NewString()[:8], safeFilename)
		diskPath := filepath.Join(uploadDir, diskFilename)

		if err := c.SaveFile(attachment, diskPath); err == nil {
			attachURL := fmt.Sprintf("/uploads/attachments/%s/%s", url.PathEscape(chatID), url.PathEscape(diskFilename))
			contentType := attachment.Header.Get("Content-Type")
			if contentType == "" {
				ext := strings.ToLower(filepath.Ext(safeFilename))
				switch ext {
				case ".pdf":
					contentType = "application/pdf"
				case ".png":
					contentType = "image/png"
				case ".jpg", ".jpeg":
					contentType = "image/jpeg"
				case ".csv":
					contentType = "text/csv"
				case ".xlsx":
					contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
				case ".xls":
					contentType = "application/vnd.ms-excel"
				default:
					contentType = "application/octet-stream"
				}
			}

			attachInfo = &models.AttachmentInfo{
				Name: safeFilename,
				URL:  attachURL,
				Type: contentType,
				Size: attachment.Size,
			}
		}
	}

	// Use client-provided message ID for dedup if present, otherwise generate one.
	msgID := c.FormValue("id")
	if msgID == "" {
		msgID = uuid.NewString()
	}
	userMsg := fiber.Map{
		"chat_id":  chatID,
		"id":       msgID,
		"user":     true,
		"user_id":  userID,
		"username": username,
		"role":     role,
		"text":     text,
		"models":   modelChoice,
	}
	if attachInfo != nil {
		userMsg["attachment"] = attachInfo.Name
		userMsg["attachment_info"] = attachInfo
	} else if attachment != nil {
		userMsg["attachment"] = attachment.Filename
	}

	userPayload, err := json.Marshal(userMsg)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to encode message")
	}
	wsHub.Broadcast(chatID, userPayload)
	persistMessage(msgID, text, modelChoice, userID, username, role, true, attachInfo, "")

	// Extract Bearer token to forward to EIAI
	token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if token == "" {
		token = "token123"
	}

	userCompany := ""
	if userVal := c.Locals("user"); userVal != nil {
		if claims, ok := userVal.(*middleware.JWTClaims); ok && claims != nil {
			userCompany = strings.TrimSpace(claims.Company)
		}
	}
	if userCompany == "" {
		userCompany = strings.TrimSpace(c.Get("X-Company"))
	}

	var resp *http.Response
	if attachment != nil {
		bodyBuf := &bytes.Buffer{}
		writer := multipart.NewWriter(bodyBuf)
		_ = writer.WriteField("query", text)
		_ = writer.WriteField("models", modelChoice)
		_ = writer.WriteField("role", role)
		_ = writer.WriteField("token", token)
		_ = writer.WriteField("chat_id", chatID)
		if userCompany != "" {
			_ = writer.WriteField("company", userCompany)
		}

		filePart, partErr := writer.CreateFormFile("attachment", attachment.Filename)
		if partErr == nil {
			src, openErr := attachment.Open()
			if openErr == nil {
				_, _ = io.Copy(filePart, src)
				src.Close()
			}
		}
		_ = writer.Close()

		req, reqErr := http.NewRequest("POST", base, bodyBuf)
		if reqErr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to create RAG request")
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)
		if userCompany != "" {
			req.Header.Set("X-Company", userCompany)
		}
		if role != "" {
			req.Header.Set("X-Role", role)
		}
		if userID != "" {
			req.Header.Set("X-User-ID", userID)
		}
		client := &http.Client{Timeout: 600 * time.Second}
		resp, err = client.Do(req)
	} else {
		params := url.Values{}
		params.Add("query", text)
		params.Add("models", modelChoice)
		params.Add("role", role)
		params.Add("token", token)
		params.Add("chat_id", chatID)
		if userCompany != "" {
			params.Add("company", userCompany)
		}

		finalURL := fmt.Sprintf("%s?%s", base, params.Encode())
		req, reqErr := http.NewRequest("GET", finalURL, nil)
		if reqErr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to create RAG request")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		if userCompany != "" {
			req.Header.Set("X-Company", userCompany)
		}
		if role != "" {
			req.Header.Set("X-Role", role)
		}
		if userID != "" {
			req.Header.Set("X-User-ID", userID)
		}
		client := &http.Client{Timeout: 600 * time.Second}
		resp, err = client.Do(req)
	}

	if err != nil {
		fmt.Printf("Error connecting to EIAI Go server: %v\n", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to contact RAG assistant")
	}
	defer resp.Body.Close()

	body2, _ := io.ReadAll(resp.Body)

	var ragResp struct {
		Answer     string                 `json:"answer"`
		Text       string                 `json:"text"`
		Attachment *models.AttachmentInfo `json:"attachment"`
	}
	var replyText string
	var aiAttach *models.AttachmentInfo

	if err := json.Unmarshal(body2, &ragResp); err == nil && (ragResp.Answer != "" || ragResp.Text != "" || ragResp.Attachment != nil) {
		replyText = ragResp.Answer
		if replyText == "" {
			replyText = ragResp.Text
		}
		aiAttach = ragResp.Attachment
	} else {
		replyText = string(body2)
	}

	aiMsgID := uuid.NewString()
	replyMsg := fiber.Map{
		"chat_id":  chatID,
		"id":       aiMsgID,
		"user":     false,
		"username": "QTERA AI",
		"role":     "system",
		"text":     replyText,
		"models":   modelChoice,
	}
	if aiAttach != nil && aiAttach.Name != "" {
		replyMsg["attachment"] = aiAttach.Name
		replyMsg["attachment_info"] = aiAttach
	}

	replyPayload, err := json.Marshal(replyMsg)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to encode message")
	}
	time.AfterFunc(700*time.Millisecond, func() {
		wsHub.Broadcast(chatID, replyPayload)
		persistMessage(aiMsgID, replyText, modelChoice, userID, username, "system", false, aiAttach, "")
	})

	return c.JSON(fiber.Map{
		"ok":   true,
		"text": text,
	})
}
