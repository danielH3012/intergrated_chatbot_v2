package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/xuri/excelize/v2"
)

// ExcelRequest payload struct for Excel report generation
type ExcelRequest struct {
	Title    string     `json:"title"`
	Filename string     `json:"filename"`
	Sheet    string     `json:"sheet"`
	Headers  []string   `json:"headers"`
	Data     [][]string `json:"data"`
}

// ExcelHandler is the Fiber HTTP handler for creating Excel (.xlsx) files
func ExcelHandler(c *fiber.Ctx) error {
	var req ExcelRequest
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
			fileName = fmt.Sprintf("%s_%d.xlsx", safeTitle, time.Now().UnixNano())
		} else {
			fileName = fmt.Sprintf("table_%d.xlsx", time.Now().UnixNano())
		}
	} else {
		fileName = filepath.Base(fileName)
		if !strings.HasSuffix(strings.ToLower(fileName), ".xlsx") {
			fileName += ".xlsx"
		}
	}

	sheetName := strings.TrimSpace(req.Sheet)
	if sheetName == "" {
		sheetName = "Asset Inventory"
	}

	filePath, err := ExcelMaker(req.Title, fileName, sheetName, req.Headers, req.Data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("failed to generate Excel file: %v", err)})
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
		"type":         "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"size":         fileSize,
		"rows":         len(req.Data),
		"columns":      len(req.Headers),
		"message":      fmt.Sprintf("Excel file generated successfully with %d rows and %d columns", len(req.Data), len(req.Headers)),
	})
}

// ExcelMaker generates a styled Excel spreadsheet saved to ./uploads
func ExcelMaker(title, fileName, sheetName string, headers []string, data [][]string) (string, error) {
	if err := os.MkdirAll("./uploads", 0755); err != nil {
		return "", err
	}

	baseName := filepath.Base(strings.TrimSpace(fileName))
	if baseName == "" || baseName == "." {
		baseName = fmt.Sprintf("table_%d.xlsx", time.Now().UnixNano())
	}
	if !strings.HasSuffix(strings.ToLower(baseName), ".xlsx") {
		baseName += ".xlsx"
	}
	filePath := filepath.Join("uploads", baseName)

	f := excelize.NewFile()
	defer f.Close()

	if sheetName == "" {
		sheetName = "Asset Inventory"
	}
	// Create custom sheet and remove default Sheet1
	sheetIndex, _ := f.NewSheet(sheetName)
	f.SetActiveSheet(sheetIndex)
	if sheetName != "Sheet1" {
		_ = f.DeleteSheet("Sheet1")
	}

	// Styles
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 10, Family: "Segoe UI"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1E293B"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#CBD5E1", Style: 1},
			{Type: "top", Color: "#CBD5E1", Style: 1},
			{Type: "right", Color: "#CBD5E1", Style: 1},
			{Type: "bottom", Color: "#CBD5E1", Style: 1},
		},
	})

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#0F172A", Size: 14, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Color: "#64748B", Size: 9, Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	dataStyleDefault, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9.5, Color: "#1E293B", Family: "Segoe UI"},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#E2E8F0", Style: 1},
			{Type: "top", Color: "#E2E8F0", Style: 1},
			{Type: "right", Color: "#E2E8F0", Style: 1},
			{Type: "bottom", Color: "#E2E8F0", Style: 1},
		},
	})

	dataStyleZebra, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9.5, Color: "#1E293B", Family: "Segoe UI"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#F8FAFC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#E2E8F0", Style: 1},
			{Type: "top", Color: "#E2E8F0", Style: 1},
			{Type: "right", Color: "#E2E8F0", Style: 1},
			{Type: "bottom", Color: "#E2E8F0", Style: 1},
		},
	})

	alignCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#E2E8F0", Style: 1},
			{Type: "top", Color: "#E2E8F0", Style: 1},
			{Type: "right", Color: "#E2E8F0", Style: 1},
			{Type: "bottom", Color: "#E2E8F0", Style: 1},
		},
	})

	alignRight, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#E2E8F0", Style: 1},
			{Type: "top", Color: "#E2E8F0", Style: 1},
			{Type: "right", Color: "#E2E8F0", Style: 1},
			{Type: "bottom", Color: "#E2E8F0", Style: 1},
		},
	})

	startRow := 1
	numCols := len(headers)
	if numCols == 0 && len(data) > 0 {
		numCols = len(data[0])
	}
	if numCols == 0 {
		numCols = 1
	}

	// 1. Document Title & Subtitle if title provided
	if strings.TrimSpace(title) != "" {
		_ = f.SetCellValue(sheetName, "A1", title)
		_ = f.SetCellStyle(sheetName, "A1", "A1", titleStyle)
		_ = f.SetRowHeight(sheetName, 1, 24)

		subtitle := fmt.Sprintf("Generated on %s | Total Records: %d", time.Now().Format("02 Jan 2006, 15:04 WIB"), len(data))
		_ = f.SetCellValue(sheetName, "A2", subtitle)
		_ = f.SetCellStyle(sheetName, "A2", "A2", subtitleStyle)
		_ = f.SetRowHeight(sheetName, 2, 18)

		startRow = 4
	}

	// Track max column lengths for auto-width calculation
	colLengths := make([]int, numCols)

	// 2. Table Headers
	if len(headers) > 0 {
		_ = f.SetRowHeight(sheetName, startRow, 26)
		for cIdx, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(cIdx+1, startRow)
			_ = f.SetCellValue(sheetName, cell, h)
			_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
			if len(h) > colLengths[cIdx] {
				colLengths[cIdx] = len(h)
			}
		}
		startRow++
	}

	// 3. Table Rows
	for rIdx, row := range data {
		rowNum := startRow + rIdx
		_ = f.SetRowHeight(sheetName, rowNum, 20)
		isZebra := (rIdx % 2) == 1

		for cIdx := 0; cIdx < numCols; cIdx++ {
			val := ""
			if cIdx < len(row) {
				val = strings.TrimSpace(row[cIdx])
			}
			cell, _ := excelize.CoordinatesToCellName(cIdx+1, rowNum)

			// Determine alignment & numeric typing
			cleaned := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(val, "Rp", ""), ".", ""), ",", ""))
			if num, err := strconv.ParseFloat(cleaned, 64); err == nil && len(cleaned) > 0 && !strings.HasPrefix(val, "0") {
				_ = f.SetCellValue(sheetName, cell, num)
				_ = f.SetCellStyle(sheetName, cell, cell, alignRight)
			} else {
				_ = f.SetCellValue(sheetName, cell, val)
				if cIdx == 0 && (len(val) <= 4 || (len(headers) > 0 && strings.EqualFold(headers[0], "no"))) {
					_ = f.SetCellStyle(sheetName, cell, cell, alignCenter)
				} else if strings.EqualFold(val, "Active") || strings.EqualFold(val, "Inactive") || strings.EqualFold(val, "Retired") {
					_ = f.SetCellStyle(sheetName, cell, cell, alignCenter)
				} else if isZebra {
					_ = f.SetCellStyle(sheetName, cell, cell, dataStyleZebra)
				} else {
					_ = f.SetCellStyle(sheetName, cell, cell, dataStyleDefault)
				}
			}

			if len(val) > colLengths[cIdx] {
				colLengths[cIdx] = len(val)
			}
		}
	}

	// 4. Auto-fit column widths
	for cIdx := 0; cIdx < numCols; cIdx++ {
		colName, _ := excelize.ColumnNumberToName(cIdx + 1)
		width := float64(colLengths[cIdx]) + 5.0
		if width < 12.0 {
			width = 12.0
		}
		if width > 45.0 {
			width = 45.0
		}
		_ = f.SetColWidth(sheetName, colName, colName, width)
	}

	if err := f.SaveAs(filePath); err != nil {
		return "", err
	}

	return filePath, nil
}