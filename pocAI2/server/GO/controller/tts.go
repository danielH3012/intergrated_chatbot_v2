package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	DefaultCartesiaVersion = "2024-11-13"
	CartesiaTTSBytesURL    = "https://api.cartesia.ai/tts/bytes"
)

type CartesiaVoice struct {
	Mode string `json:"mode"`
	ID   string `json:"id"`
}

type CartesiaOutputFormat struct {
	Container  string `json:"container"`
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
}

type CartesiaTTSRequest struct {
	ModelID      string               `json:"model_id"`
	Transcript   string               `json:"transcript"`
	Voice        CartesiaVoice        `json:"voice"`
	OutputFormat CartesiaOutputFormat `json:"output_format"`
	Language     string               `json:"language,omitempty"`
}

// CleanTextForTTS removes markdown, code blocks, links, and system artifacts for natural speech synthesis.
func CleanTextForTTS(text string) string {
	if text == "" {
		return ""
	}

	// Remove attachment indicators
	reAttachment := regexp.MustCompile(`(?i)\n?📎\s*\[Attached:\s*[^\]]+\]`)
	text = reAttachment.ReplaceAllString(text, "")

	// Remove fenced code blocks
	reCodeBlock := regexp.MustCompile("(?s)```.*?```")
	text = reCodeBlock.ReplaceAllString(text, "")

	// Remove inline code
	reInlineCode := regexp.MustCompile("`([^`]+)`")
	text = reInlineCode.ReplaceAllString(text, "$1")

	// Remove markdown links [text](url) -> text
	reLinks := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	text = reLinks.ReplaceAllString(text, "$1")

	// Remove markdown formatting characters (*, #, _, ~, >, ===)
	reSymbols := regexp.MustCompile(`[*#_~>=]`)
	text = reSymbols.ReplaceAllString(text, "")

	// Normalize excessive whitespace
	reSpaces := regexp.MustCompile(`\s+`)
	text = reSpaces.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// DetectLanguageTTS detects if the text is primarily Indonesian or English.
func DetectLanguageTTS(text string) string {
	lower := strings.ToLower(text)
	indonesianKeywords := []string{
		"yang", "dan", "di", "ini", "itu", "ada", "tidak", "untuk", "dengan",
		"dari", "pada", "ke", "aset", "adalah", "sudah", "bisa", "perusahaan", "pengguna",
	}
	for _, kw := range indonesianKeywords {
		if strings.Contains(lower, " "+kw+" ") || strings.HasPrefix(lower, kw+" ") || strings.HasSuffix(lower, " "+kw) {
			return "id"
		}
	}
	return "en"
}

// SynthesizeCartesiaSonic calls the Cartesia REST API with Sonic 3.6 to generate speech audio bytes.
func SynthesizeCartesiaSonic(text, voiceID, lang string) ([]byte, error) {
	apiKey := os.Getenv("CARTESIA_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("CARTESIA_KEY is not configured in environment")
	}

	modelID := os.Getenv("CARTESIA_MODEL")
	if modelID == "" {
		return nil, fmt.Errorf("CARTESIA_MODEL is not configured in environment")
	}

	if voiceID == "" {
		voiceID = os.Getenv("CARTESIA_VOICE_ID")
		if voiceID == "" {
			return nil, fmt.Errorf("CARTESIA_VOICE_ID is not configured in environment")
		}
	}

	cleanTranscript := CleanTextForTTS(text)
	if cleanTranscript == "" {
		return nil, fmt.Errorf("transcript is empty after cleaning")
	}

	if lang == "" {
		lang = DetectLanguageTTS(cleanTranscript)
	}

	apiVersion := os.Getenv("CARTESIA_VERSION")
	if apiVersion == "" {
		apiVersion = DefaultCartesiaVersion
	}

	reqBody := CartesiaTTSRequest{
		ModelID:    modelID,
		Transcript: cleanTranscript,
		Voice: CartesiaVoice{
			Mode: "id",
			ID:   voiceID,
		},
		OutputFormat: CartesiaOutputFormat{
			Container:  "wav",
			Encoding:   "pcm_s16le",
			SampleRate: 44100,
		},
		Language: lang,
	}

	payloadBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Cartesia request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", CartesiaTTSBytesURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create Cartesia HTTP request: %w", err)
	}

	httpReq.Header.Set("X-API-Key", apiKey)
	httpReq.Header.Set("Cartesia-Version", apiVersion)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request to Cartesia API failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Cartesia response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cartesia API error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	log.Printf("[Cartesia TTS] Successfully generated %d bytes of audio (model: %s, voice: %s, lang: %s)",
		len(bodyBytes), modelID, voiceID, lang)

	return bodyBytes, nil
}

// TTSHandler handles HTTP requests to synthesize text-to-speech using Cartesia Sonic 3.6.
// Supports both POST (JSON or form-urlencoded) and GET (query params).
func TTSHandler(c *fiber.Ctx) error {
	var text, voiceID, lang string

	if c.Method() == fiber.MethodGet {
		text = c.Query("text")
		voiceID = c.Query("voice_id")
		lang = c.Query("language")
	} else {
		var req struct {
			Text     string `json:"text" form:"text"`
			VoiceID  string `json:"voice_id" form:"voice_id"`
			Language string `json:"language" form:"language"`
		}
		if err := c.BodyParser(&req); err == nil {
			text = req.Text
			voiceID = req.VoiceID
			lang = req.Language
		}
		if text == "" {
			text = c.FormValue("text")
		}
		if voiceID == "" {
			voiceID = c.FormValue("voice_id")
		}
		if lang == "" {
			lang = c.FormValue("language")
		}
	}

	if strings.TrimSpace(text) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "text parameter is required",
		})
	}

	audioBytes, err := SynthesizeCartesiaSonic(text, voiceID, lang)
	if err != nil {
		log.Printf("[TTSHandler] Error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed to generate speech",
			"details": err.Error(),
		})
	}

	c.Set("Content-Type", "audio/wav")
	c.Set("Content-Disposition", `inline; filename="speech.wav"`)
	c.Set("Cache-Control", "public, max-age=3600")
	return c.Send(audioBytes)
}
