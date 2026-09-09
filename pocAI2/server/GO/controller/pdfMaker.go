package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/gofiber/fiber/v2"
)

// Request payload struct if client sends custom data
type PdfRequest struct {
	Title   string     `json:"title"`
	Headers []string   `json:"headers"`
	Data    [][]string `json:"data"`
}

// PdfHandler is the Fiber HTTP handler
func PdfHandler(c *fiber.Ctx) error {
	var req PdfRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if len(req.Headers) == 0 && len(req.Data) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "headers and data are required"})
	}

	filePath := PdfMaker(req.Title, req.Data, req.Headers)

	// Direct download if query param ?download=true is passed
	if c.Query("download") == "true" {
		return c.Download(filePath)
	}

	// JSON response for MCP and API clients
	relPath := "/" + filepath.ToSlash(filePath)
	var fileSize int64
	if fi, err := os.Stat(filePath); err == nil {
		fileSize = fi.Size()
	}

	return c.JSON(fiber.Map{
		"ok":           true,
		"filename":     filepath.Base(filePath),
		"download_url": relPath,
		"type":         "application/pdf",
		"size":         fileSize,
		"rows":         len(req.Data),
		"columns":      len(req.Headers),
		"message":      fmt.Sprintf("PDF generated successfully with %d rows and %d columns", len(req.Data), len(req.Headers)),
	})
}

func addWatermark(pdf *fpdf.Fpdf, text string, pageW, pageH float64) {
	pdf.SetFont("Arial", "B", 70)
	pdf.SetTextColor(100, 116, 139) // slate-500

	// Enable semi-transparency so watermark is clearly visible over content
	pdf.SetAlpha(0.18, "Normal")

	// Rotate & center watermark across page
	pdf.TransformBegin()
	pdf.TransformRotate(35, pageW/2, pageH/2)
	pdf.SetXY(0, pageH/2-20)
	pdf.CellFormat(pageW, 40, text, "", 0, "C", false, 0, "")
	pdf.TransformEnd()

	// Reset opacity
	pdf.SetAlpha(1.0, "Normal")
}

// fitText truncates text with ellipsis if it exceeds the cell's printable width.
func fitText(pdf *fpdf.Fpdf, text string, maxWidth float64) string {
	if maxWidth <= 4.0 {
		return text
	}
	targetWidth := maxWidth - 2.5 // padding margin
	if pdf.GetStringWidth(text) <= targetWidth {
		return text
	}
	runes := []rune(text)
	for len(runes) > 1 && pdf.GetStringWidth(string(runes)+"…") > targetWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func PdfMaker(title string, data [][]string, header []string) string {
	_ = os.MkdirAll("./uploads", 0755)
	filename := fmt.Sprintf("table_%d.pdf", time.Now().UnixNano())
	filePath := filepath.Join("uploads", filename)

	numCols := len(header)
	if numCols == 0 {
		numCols = 1
	}

	// Automatically choose Landscape if table has 6 or more columns for clean layout
	orientation := "P"
	pageW, pageH := 210.0, 297.0 // A4 Portrait (mm)
	if numCols >= 6 {
		orientation = "L"
		pageW, pageH = 297.0, 210.0 // A4 Landscape (mm)
	}

	margin := 12.0
	usableWidth := pageW - (margin * 2)

	pdf := fpdf.New(orientation, "mm", "A4", "")
	pdf.SetMargins(margin, margin, margin)
	pdf.SetAutoPageBreak(true, margin)
	pdf.AddPage()

	// Set cursor position to top-left margin
	pdf.SetXY(margin, margin)

	// 2. Document Title Header
	if strings.TrimSpace(title) == "" {
		title = "Asset Inventory Report"
	}
	pdf.SetFont("Arial", "B", 15)
	pdf.SetTextColor(30, 41, 59) // slate-800
	pdf.CellFormat(usableWidth, 9, title, "", 1, "L", false, 0, "")

	// Subtitle with timestamp & summary
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(100, 116, 139) // slate-500
	subtitle := fmt.Sprintf("Generated on %s  |  Total Records: %d", time.Now().Format("02 Jan 2006, 15:04 WIB"), len(data))
	pdf.CellFormat(usableWidth, 6, subtitle, "", 1, "L", false, 0, "")
	pdf.Ln(3)

	// 3. Dynamic Column Width Calculation based on content
	colWidths := make([]float64, numCols)
	minWidths := make([]float64, numCols)

	// Sample fonts for measuring
	pdf.SetFont("Arial", "B", 9)
	for i, h := range header {
		minWidths[i] = pdf.GetStringWidth(h) + 6.0
	}

	pdf.SetFont("Arial", "", 8.5)
	for _, row := range data {
		for i, val := range row {
			if i < numCols {
				w := pdf.GetStringWidth(val) + 5.0
				if w > minWidths[i] {
					minWidths[i] = w
				}
			}
		}
	}

	// Cap individual column widths to prevent single long strings from dominating
	maxAllowedColWidth := usableWidth * 0.35
	totalMeasured := 0.0
	for i := range minWidths {
		if minWidths[i] < 12.0 {
			minWidths[i] = 12.0
		}
		if minWidths[i] > maxAllowedColWidth {
			minWidths[i] = maxAllowedColWidth
		}
		totalMeasured += minWidths[i]
	}

	// Scale proportionally to fill exact usable page width
	scale := usableWidth / totalMeasured
	for i := range colWidths {
		colWidths[i] = minWidths[i] * scale
	}

	// 4. Render Table Header
	headerFontSize := 9.0
	if numCols >= 8 {
		headerFontSize = 8.0
	}

	renderHeader := func() {
		pdf.SetFont("Arial", "B", headerFontSize)
		pdf.SetFillColor(30, 41, 59)    // Slate-800 dark header
		pdf.SetTextColor(255, 255, 255) // White text
		pdf.SetDrawColor(203, 213, 225) // Slate-300 borders
		pdf.SetLineWidth(0.2)

		for i, h := range header {
			text := fitText(pdf, h, colWidths[i])
			pdf.CellFormat(colWidths[i], 8.5, text, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)
	}

	renderHeader()

	// 5. Render Data Rows with Dynamic Multi-line Text Wrapping (like Excel)
	dataFontSize := 8.5
	lineHeight := 4.2
	if numCols >= 8 {
		dataFontSize = 7.5
		lineHeight = 3.8
	}
	cellPadX := 2.0
	cellPadY := 1.8

	// Disable auto page break during table row drawing; we handle page breaks cleanly per row
	pdf.SetAutoPageBreak(false, margin)

	fill := false
	for _, row := range data {
		pdf.SetFont("Arial", "", dataFontSize)

		// A. Pre-split all cells in this row and calculate required max lines
		rowLines := make([][]string, numCols)
		maxLines := 1
		for i := 0; i < numCols; i++ {
			val := ""
			if i < len(row) {
				val = strings.TrimSpace(row[i])
			}
			w := colWidths[i] - (cellPadX * 2)
			if w < 2.0 {
				w = 2.0
			}
			lines := pdf.SplitText(val, w)
			if len(lines) == 0 {
				lines = []string{""}
			}
			rowLines[i] = lines
			if len(lines) > maxLines {
				maxLines = len(lines)
			}
		}

		rowHeight := float64(maxLines)*lineHeight + (cellPadY * 2)
		if rowHeight < 7.5 {
			rowHeight = 7.5
		}

		// B. Page break check: if current row exceeds page height, start new page & re-draw header
		if pdf.GetY()+rowHeight > pageH-margin {
			pdf.AddPage()
			renderHeader()
			pdf.SetFont("Arial", "", dataFontSize)
		}

		startY := pdf.GetY()
		startX := margin

		if fill {
			pdf.SetFillColor(248, 250, 252) // slate-50 zebra stripe
		} else {
			pdf.SetFillColor(255, 255, 255) // white
		}
		pdf.SetDrawColor(203, 213, 225) // slate-300 cell borders
		pdf.SetTextColor(30, 41, 59)    // slate-800 text

		// C. Render each cell background, border, and vertically centered wrapped lines
		for i := 0; i < numCols; i++ {
			cWidth := colWidths[i]
			pdf.Rect(startX, startY, cWidth, rowHeight, "FD")

			lines := rowLines[i]
			align := "L"
			val := ""
			if i < len(row) {
				val = strings.TrimSpace(row[i])
			}

			if i == 0 && (len(val) <= 3 || strings.EqualFold(header[0], "no")) {
				align = "C"
			} else if strings.EqualFold(val, "Active") || strings.EqualFold(val, "Inactive") || strings.EqualFold(val, "Retired") {
				align = "C"
			} else {
				cleaned := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(val, "Rp", ""), ".", ""), ",", ""))
				if _, err := strconv.ParseFloat(cleaned, 64); err == nil && len(cleaned) > 0 {
					align = "R"
				}
			}

			totalTextH := float64(len(lines)) * lineHeight
			offsetY := (rowHeight - totalTextH) / 2.0

			for lIdx, line := range lines {
				lineY := startY + offsetY + (float64(lIdx) * lineHeight)
				pdf.SetXY(startX+cellPadX, lineY)
				pdf.CellFormat(cWidth-(cellPadX*2), lineHeight, line, "", 0, align, false, 0, "")
			}

			startX += cWidth
		}

		pdf.SetXY(margin, startY+rowHeight)
		fill = !fill
	}

	// Render watermark overlay on ALL pages after content is rendered
	totalPages := pdf.PageCount()
	for p := 1; p <= totalPages; p++ {
		pdf.SetPage(p)
		addWatermark(pdf, "QTERA", pageW, pageH)
	}

	_ = pdf.OutputFileAndClose(filePath)
	return filePath
}
