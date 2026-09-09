package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang-dh/middleware"
)

type ElevenLabsScribeResponse struct {
	LanguageCode string `json:"language_code"`
	Text         string `json:"text"`
}

// preprocessAudio cleans background noise, removes DC rumble,
// normalizes vocal loudness, and trims dead silence using ffmpeg.
func preprocessAudio(inputPath string) (string, func()) {
	_ = os.MkdirAll("./tmp", 0755)
	cleanedPath := filepath.Join("./tmp", fmt.Sprintf("clean_%s.wav", uuid.NewString()))
	cleanup := func() { _ = os.Remove(cleanedPath) }

	// 1. highpass: eliminates mic thumps and low-frequency rumble below 80Hz
	// 2. lowpass: removes electronic hiss and high-frequency noise above 8000Hz
	// 3. afftdn: reduces ambient stationary noise (fan, AC, room hum)
	// 4. loudnorm: boosts quiet speech and balances volume to broadcast standard
	// 5. silenceremove: strips leading and trailing silence
	filterChain := "highpass=f=80,lowpass=f=8000,afftdn=nf=-25,loudnorm=I=-16:TP=-1.5:LRA=11,silenceremove=start_periods=1:start_duration=0.05:start_threshold=-50dB:stop_periods=-1:stop_duration=0.8:stop_threshold=-50dB"

	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, "-af", filterChain, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", cleanedPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[preprocessAudio] ffmpeg filter notice: %v (%s). Using original audio.", err, strings.TrimSpace(string(out)))
		return inputPath, func() {}
	}

	if fi, err := os.Stat(cleanedPath); err == nil && fi.Size() > 1024 {
		log.Printf("[preprocessAudio] Cleaned audio generated (%d bytes)", fi.Size())
		return cleanedPath, cleanup
	}

	return inputPath, func() {}
}

// TranscribeAudio transcribes an audio file using ElevenLabs Scribe v2 API.
func TranscribeAudio(audioPath string) (string, error) {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("TTS_API")
	}
	if apiKey == "" {
		return "", fmt.Errorf("ELEVENLABS_API_KEY is not configured")
	}

	modelID := os.Getenv("ELEVENLABS_SCRIBE_MODEL")
	if modelID == "" {
		modelID = "scribe_v2"
	}

	// Clean audio background noise and boost speech clarity
	cleanPath, cleanup := preprocessAudio(audioPath)
	defer cleanup()
	audioPath = cleanPath

	file, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	filename := filepath.Base(audioPath)
	if filepath.Ext(filename) == "" {
		filename = "audio.wav"
	}

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create multipart form: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}

	_ = writer.WriteField("model_id", modelID)
	if lang := os.Getenv("ELEVENLABS_LANGUAGE_CODE"); lang != "" {
		_ = writer.WriteField("language_code", lang)
	}
	_ = writer.Close()

	req, err := http.NewRequest("POST", "https://api.elevenlabs.io/v1/speech-to-text", body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request to ElevenLabs failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ElevenLabs API error (%d): %s", resp.StatusCode, string(respBytes))
	}

	var scribeResp ElevenLabsScribeResponse
	if err := json.Unmarshal(respBytes, &scribeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	result := strings.TrimSpace(scribeResp.Text)

	// Suppress non-speech audio events (e.g. [beep], [silence], etc.)
	if (strings.HasPrefix(result, "(") && strings.HasSuffix(result, ")")) ||
		(strings.HasPrefix(result, "[") && strings.HasSuffix(result, "]")) {
		lower := strings.ToLower(result)
		if strings.Contains(lower, "blank") ||
			strings.Contains(lower, "silence") ||
			strings.Contains(lower, "beep") ||
			strings.Contains(lower, "cricket") ||
			strings.Contains(lower, "applause") ||
			strings.Contains(lower, "music") {
			result = ""
		}
	}

	log.Printf("[ElevenLabs Scribe] Lang: %s, Text: %q", scribeResp.LanguageCode, result)
	return result, nil
}

// TranscribeLocal is an alias for TranscribeAudio for backwards compatibility
func TranscribeLocal(audioPath string) (string, error) {
	return TranscribeAudio(audioPath)
}

// SendAudioMessage handles multipart audio file uploads, transcribes speech with Scribe v2, and dispatches to RAG.
func SendAudioMessage(c *fiber.Ctx) error {
	file, err := c.FormFile("audio")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "audio file is required"})
	}

	chatID := c.FormValue("chat_id")
	if chatID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "chat_id is required"})
	}

	modelChoice := c.FormValue("models")
	if modelChoice == "" {
		modelChoice = "qwen"
	}

	_ = os.MkdirAll("./tmp", 0755)
	tempAudioPath := filepath.Join("./tmp", fmt.Sprintf("audio_%s_%s", uuid.NewString(), file.Filename))
	if err := c.SaveFile(file, tempAudioPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save audio file"})
	}
	defer os.Remove(tempAudioPath)

	transcription, err := TranscribeAudio(tempAudioPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "transcription failed",
			"details": err.Error(),
		})
	}

	if strings.TrimSpace(transcription) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no speech detected in audio"})
	}

	// Resolve user identity from JWT context
	userID := "anonymous"
	username := "Anonymous"
	role := "operator"
	if userVal := c.Locals("user"); userVal != nil {
		if claims, ok := userVal.(*middleware.JWTClaims); ok && claims != nil {
			userID = claims.UserID
			username = claims.Username
			role = claims.Role
		}
	}

	return ProcessAndDispatchMessage(c, chatID, transcription, modelChoice, userID, username, role)
}

// TranscribeAudioHandler transcribes uploaded audio and returns the text for user review/editing in the input box.
func TranscribeAudioHandler(c *fiber.Ctx) error {
	file, err := c.FormFile("audio")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "audio file is required"})
	}

	_ = os.MkdirAll("./tmp", 0755)
	tempAudioPath := filepath.Join("./tmp", fmt.Sprintf("audio_%s_%s", uuid.NewString(), file.Filename))
	if err := c.SaveFile(file, tempAudioPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save audio file"})
	}
	defer os.Remove(tempAudioPath)

	transcription, err := TranscribeAudio(tempAudioPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "transcription failed",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"ok":   true,
		"text": transcription,
	})
}
