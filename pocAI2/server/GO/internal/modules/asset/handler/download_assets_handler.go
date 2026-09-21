package handler

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/xuri/excelize/v2"

	"golang-dh/internal/modules/asset/dto"
	"golang-dh/pkg/errs"
)

// HandleDownloadAssets handles GET /api/assets/download.
func (h *AssetHandler) HandleDownloadAssets(c *fiber.Ctx) error {
	_, company := extractUserIdentity(c)
	if company == "" {
		return errs.BadRequest("Company is required to download assets").SendResponse(c)
	}

	filter := dto.AssetListFilter{
		Perusahaan: company,
		Search:     strings.TrimSpace(c.Query("search")),
		Category:   strings.TrimSpace(c.Query("category")),
		Brand:      strings.TrimSpace(c.Query("brand")),
		Location:   strings.TrimSpace(c.Query("location")),
		Status:     strings.TrimSpace(c.Query("status")),
		Sort:       strings.TrimSpace(c.Query("sort", "createdAt")),
		Order:      strings.TrimSpace(c.Query("order", "DESC")),
		AllRecords: true,
	}

	result, err := h.svc.ListAssets(c.Context(), filter)
	if err != nil {
		return errs.InternalServerError("Failed to fetch assets for download", err.Error()).SendResponse(c)
	}

	format := strings.ToLower(c.Query("format", "csv"))
	if format == "xlsx" || format == "excel" {
		return streamExcel(c, result.Assets)
	}
	return streamCSV(c, result.Assets)
}

func streamCSV(c *fiber.Ctx, assets []dto.AssetItem) error {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	headers := []string{"Asset ID", "Name", "Category", "Brand", "Model/Type", "Purchase Date", "Purchase Price", "Location", "Created At", "Company", "Status"}
	_ = writer.Write(headers)

	for _, a := range assets {
		row := []string{a.AssetID, a.Name, a.Category, a.Brand, a.ModelType, a.PurchaseDate, a.PurchasePrice, a.Location, a.CreatedAt, a.Perusahaan, a.Status}
		_ = writer.Write(row)
	}
	writer.Flush()

	filename := fmt.Sprintf("Assets_%s.csv", time.Now().Format("20060102_150405"))
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	return c.Send(buf.Bytes())
}

func streamExcel(c *fiber.Ctx, assets []dto.AssetItem) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Assets"
	f.SetSheetName("Sheet1", sheet)

	headers := []any{"Asset ID", "Name", "Category", "Brand", "Model/Type", "Purchase Date", "Purchase Price", "Location", "Created At", "Company", "Status"}
	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}

	for rowIdx, a := range assets {
		r := rowIdx + 2
		vals := []any{a.AssetID, a.Name, a.Category, a.Brand, a.ModelType, a.PurchaseDate, a.PurchasePrice, a.Location, a.CreatedAt, a.Perusahaan, a.Status}
		for colIdx, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, r)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return errs.InternalServerError("Failed to build Excel spreadsheet", err.Error()).SendResponse(c)
	}

	filename := fmt.Sprintf("Assets_%s.xlsx", time.Now().Format("20060102_150405"))
	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	return c.Send(buf.Bytes())
}
