package controller

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// SynthesizeAnomalyNarrative calls the AI gateway to weave multiple anomaly signals.
// If AI fails or is not configured, it returns an error and empty narrative (no placeholders or fake templates).
func SynthesizeAnomalyNarrative(borrowerName, groupName string, signals []string) (string, error) {
	if len(signals) == 0 {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	prompt := fmt.Sprintf(
		"Sebagai asisten AI approval peminjaman aset, rangkailah temuan sinyal anomali ini menjadi satu kalimat bahasa Indonesia yang mengalir, jelas, dan profesional untuk approver:\nPeminjam: %s (Group: %s)\nSinyal Anomali:\n- %s\nBerikan HANYA satu kalimat narasi langsung tanpa pengantar.",
		borrowerName, groupName, strings.Join(signals, "\n- "),
	)

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	log.Printf("[anomaly_narrative] Requesting AI narrative for %s (Group: %s) with %d signals...", borrowerName, groupName, len(signals))
	res, err := ChatGenerate(ctx, messages, nil, 1024, GeneratorModel, 2)
	if err != nil {
		log.Printf("[anomaly_narrative] AI generation failed: %v", err)
		return "", fmt.Errorf("AI Gateway Error: %v", err)
	}
	if res == nil || strings.TrimSpace(res.RawOutput) == "" {
		log.Printf("[anomaly_narrative] AI returned empty output")
		return "", fmt.Errorf("AI Gateway returned empty response")
	}

	cleaned := strings.TrimSpace(res.RawOutput)
	cleaned = strings.Trim(cleaned, "\"")
	log.Printf("[anomaly_narrative] AI Narrative generated: %s", cleaned)
	return cleaned, nil
}
