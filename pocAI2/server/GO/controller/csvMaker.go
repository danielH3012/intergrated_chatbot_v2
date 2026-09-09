package controllers

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CsvRequest payload struct for CSV generation
type CsvRequest struct {
	Title    string     `json:"title"`
	Filename string     `json:"filename"`
	Headers  []string   `json:"headers"`
	Data     [][]string `json:"data"`
}

// CsvHandler is the Fiber HTTP handler for creating CSV files
func CsvHandler(c *fiber.Ctx) error {
	var req CsvRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if len(req.Headers) == 0 && len(req.Data) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "headers and data are required"})
	}

	fileName := strings.TrimSpace(req.Filename)
	if fileName == "" {
		if strings.TrimSpace(req.Title) != "" {
			safeTitle := sanitizeFilename(req.Title)
			fileName = fmt.Sprintf("%s_%d.csv", safeTitle, time.Now().UnixNano())
		} else {
			fileName = fmt.Sprintf("table_%d.csv", time.Now().UnixNano())
		}
	} else {
		fileName = filepath.Base(fileName)
		if !strings.HasSuffix(strings.ToLower(fileName), ".csv") {
			fileName += ".csv"
		}
	}

	filePath, err := CsvMaker(req.Data, req.Headers, fileName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("failed to generate CSV: %v", err)})
	}

	// Direct download if query param ?download=true is passed
	if c.Query("download") == "true" {
		return c.Download(filePath)
	}

	relPath := "/" + filepath.ToSlash(filePath)
	var fileSize int64
	if fi, err := os.Stat(filePath); err == nil {
		fileSize = fi.Size()
	}

	return c.JSON(fiber.Map{
		"ok":           true,
		"filename":     filepath.Base(filePath),
		"download_url": relPath,
		"type":         "text/csv",
		"size":         fileSize,
		"rows":         len(req.Data),
		"columns":      len(req.Headers),
		"message":      fmt.Sprintf("CSV generated successfully with %d rows and %d columns", len(req.Data), len(req.Headers)),
	})
}

// CsvMaker writes headers and rows to a CSV file in ./uploads and returns its path safely.
func CsvMaker(rows [][]string, header []string, fileName string) (string, error) {
	if err := os.MkdirAll("./uploads", 0755); err != nil {
		return "", err
	}

	baseName := filepath.Base(strings.TrimSpace(fileName))
	if baseName == "" || baseName == "." {
		baseName = fmt.Sprintf("table_%d.csv", time.Now().UnixNano())
	}
	if !strings.HasSuffix(strings.ToLower(baseName), ".csv") {
		baseName += ".csv"
	}

	filePath := filepath.Join("uploads", baseName)
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// UTF-8 BOM for Excel / spreadsheet compatibility
	_, _ = file.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if provided
	if len(header) > 0 {
		if err := writer.Write(header); err != nil {
			return "", err
		}
	}

	// Write data rows
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return filePath, nil
}

func sanitizeFilename(name string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	safe := reg.ReplaceAllString(strings.TrimSpace(name), "_")
	safe = strings.Trim(safe, "_")
	if safe == "" {
		return "export"
	}
	if len(safe) > 50 {
		return safe[:50]
	}
	return safe
}
