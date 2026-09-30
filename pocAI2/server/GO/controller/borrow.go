package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// BorrowApprovalItem represents an item in an approval transaction
type BorrowApprovalItem struct {
	ID                   string  `json:"id"`
	TransactionID        string  `json:"transaction_id"`
	AssetID              string  `json:"asset_id"`
	AssetName            string  `json:"asset_name"`
	Brand                string  `json:"brand"`
	ModelType            string  `json:"model_type"`
	Category             string  `json:"category"`
	DurationDays         int     `json:"duration_days"`
	ApprovedDurationDays *int    `json:"approved_duration_days"`
	Note                 *string `json:"note"`
	IsApproved           *bool   `json:"is_approved"`
	Status               string  `json:"status"`
}

// BorrowApprovalTransaction represents header + items for approval
type BorrowApprovalTransaction struct {
	ID              string               `json:"id"`
	TransactionCode string               `json:"transaction_code"`
	BorrowerID      string               `json:"borrower_id"`
	BorrowerName    string               `json:"borrower_name"`
	ManagerID       string               `json:"manager_id"`
	ManagerName     string               `json:"manager_name"`
	GroupName       string               `json:"group_name"`
	RequestDate     time.Time            `json:"request_date"`
	Status          string               `json:"status"`
	ItemCount       int                  `json:"item_count"`
	Items           []BorrowApprovalItem `json:"items,omitempty"`
}

// AnomalyContextItem represents a metric comparison
type AnomalyContextItem struct {
	Metric   string `json:"metric"`
	GroupVal int    `json:"group_val"`
	TotalVal int    `json:"total_val"`
	Status   string `json:"status"` // "normal" or "menonjol"
}

// AnomalySignal represents a single detected anomaly signal
type AnomalySignal struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Triggered   bool   `json:"triggered"`
	Description string `json:"description"`
}

// AnomalyReportResponse is the response format for Check Anomalies
type AnomalyReportResponse struct {
	DataAsOf          string               `json:"data_as_of"`
	TransactionCode   string               `json:"transaction_code"`
	BorrowerName      string               `json:"borrower_name"`
	GroupName         string               `json:"group_name"`
	BorrowerContext   []AnomalyContextItem `json:"borrower_context"`
	RequestContext    []AnomalySignal      `json:"request_context"`
	SignalsCount      int                  `json:"signals_count"`
	ReviewRecommended bool                 `json:"review_recommended"`
	Narrative         string               `json:"narrative"`
	ErrorMessage      string               `json:"error_message,omitempty"`
	IsAIGenerated     bool                 `json:"is_ai_generated"`
}

// GetBorrowApprovals handles GET /api/borrow/approvals
func GetBorrowApprovals(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := DB.Query(ctx, `
		SELECT t.id, t.transaction_code, t.borrower_id, t.borrower_name, 
		       COALESCE(t.manager_id, ''), t.manager_name, t.group_name, 
		       t.request_date, t.status, COUNT(i.id) AS item_count
		FROM public.borrow_transactions t
		LEFT JOIN public.borrow_items i ON t.id = i.transaction_id
		WHERE t.status = 'pending_approval'
		GROUP BY t.id, t.transaction_code, t.borrower_id, t.borrower_name, 
		         t.manager_id, t.manager_name, t.group_name, t.request_date, t.status
		ORDER BY t.request_date DESC
	`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	var result []BorrowApprovalTransaction
	for rows.Next() {
		var item BorrowApprovalTransaction
		if err := rows.Scan(
			&item.ID, &item.TransactionCode, &item.BorrowerID, &item.BorrowerName,
			&item.ManagerID, &item.ManagerName, &item.GroupName,
			&item.RequestDate, &item.Status, &item.ItemCount,
		); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		result = append(result, item)
	}

	return c.JSON(fiber.Map{"transactions": result})
}

// GetBorrowTransaction handles GET /api/borrow/:id
func GetBorrowTransaction(c *fiber.Ctx) error {
	idParam := c.Params("id")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var trx BorrowApprovalTransaction
	err := DB.QueryRow(ctx, `
		SELECT id, transaction_code, borrower_id, borrower_name, 
		       COALESCE(manager_id, ''), manager_name, group_name, 
		       request_date, status
		FROM public.borrow_transactions
		WHERE id::text = $1 OR transaction_code = $1
	`, idParam).Scan(
		&trx.ID, &trx.TransactionCode, &trx.BorrowerID, &trx.BorrowerName,
		&trx.ManagerID, &trx.ManagerName, &trx.GroupName,
		&trx.RequestDate, &trx.Status,
	)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Transaction not found"})
	}

	rows, err := DB.Query(ctx, `
		SELECT i.id, i.transaction_id, i.asset_id, COALESCE(a.name, ''), 
		       COALESCE(a.brand, ''), COALESCE(a.model_type, ''), COALESCE(a.category, ''),
		       i.duration_days, i.approved_duration_days, i.note, i.is_approved, i.status
		FROM public.borrow_items i
		LEFT JOIN public.assets a ON i.asset_id = a.asset_id
		WHERE i.transaction_id = $1
	`, trx.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var itm BorrowApprovalItem
			_ = rows.Scan(
				&itm.ID, &itm.TransactionID, &itm.AssetID, &itm.AssetName,
				&itm.Brand, &itm.ModelType, &itm.Category,
				&itm.DurationDays, &itm.ApprovedDurationDays, &itm.Note, &itm.IsApproved, &itm.Status,
			)
			trx.Items = append(trx.Items, itm)
		}
	}
	trx.ItemCount = len(trx.Items)

	return c.JSON(trx)
}

// CheckBorrowAnomalies handles GET /api/borrow/:id/anomalies
func CheckBorrowAnomalies(c *fiber.Ctx) error {
	idParam := c.Params("id")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	// 1. Get Transaction Info
	var trx BorrowApprovalTransaction
	err := DB.QueryRow(ctx, `
		SELECT id, transaction_code, borrower_id, borrower_name, group_name, request_date
		FROM public.borrow_transactions
		WHERE id::text = $1 OR transaction_code = $1
	`, idParam).Scan(
		&trx.ID, &trx.TransactionCode, &trx.BorrowerID, &trx.BorrowerName, &trx.GroupName, &trx.RequestDate,
	)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Transaction not found"})
	}

	// 2. Fetch Items
	rows, err := DB.Query(ctx, `
		SELECT i.id, i.asset_id, COALESCE(a.name, ''), COALESCE(a.category, ''), i.duration_days
		FROM public.borrow_items i
		LEFT JOIN public.assets a ON i.asset_id = a.asset_id
		WHERE i.transaction_id = $1
	`, trx.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	type itemDetail struct {
		AssetID      string
		AssetName    string
		Category     string
		DurationDays int
	}
	var items []itemDetail
	for rows.Next() {
		var itm itemDetail
		var dummyID string
		_ = rows.Scan(&dummyID, &itm.AssetID, &itm.AssetName, &itm.Category, &itm.DurationDays)
		items = append(items, itm)
	}

	// 3. Compute 7 Signals
	var triggeredSignals []string
	var allSignals []AnomalySignal

	// Sinyal 1: Durasi tiap aset vs kebiasaan kategori (Ambang: > 28 hari atau > 2x default borrow time)
	s1Triggered := false
	s1Desc := "Durasi request sesuai batas normal kebiasaan kategori aset."
	for _, it := range items {
		if it.DurationDays > 28 { // > 2x 14 hari
			s1Triggered = true
			s1Desc = fmt.Sprintf("Durasi pinjam (%d hari) melampaui batas kebiasaan kategori %s (>2x default borrow time).", it.DurationDays, it.Category)
			triggeredSignals = append(triggeredSignals, s1Desc)
			break
		}
	}
	allSignals = append(allSignals, AnomalySignal{
		ID:          1,
		Name:        "Durasi tiap aset vs kebiasaan kategori",
		Triggered:   s1Triggered,
		Description: s1Desc,
	})

	// Sinyal 2: Pinjaman aktif peminjam (Group ini & Total)
	var activeLoansGroup, activeLoansTotal int
	_ = DB.QueryRow(ctx, `
		SELECT COUNT(DISTINCT t.id)
		FROM public.borrow_transactions t
		WHERE t.borrower_id = $1 AND t.status = 'active' AND t.group_name = $2
	`, trx.BorrowerID, trx.GroupName).Scan(&activeLoansGroup)

	_ = DB.QueryRow(ctx, `
		SELECT COUNT(DISTINCT t.id)
		FROM public.borrow_transactions t
		WHERE t.borrower_id = $1 AND t.status = 'active'
	`, trx.BorrowerID).Scan(&activeLoansTotal)

	s2Triggered := activeLoansTotal >= 3
	s2Desc := fmt.Sprintf("Peminjam saat ini memegang %d pinjaman aktif.", activeLoansTotal)
	if s2Triggered {
		s2Desc = fmt.Sprintf("Peminjam memiliki %d pinjaman aktif sekaligus (ambang batas >= 3).", activeLoansTotal)
		triggeredSignals = append(triggeredSignals, s2Desc)
	}
	allSignals = append(allSignals, AnomalySignal{
		ID:          2,
		Name:        "Pinjaman aktif peminjam",
		Triggered:   s2Triggered,
		Description: s2Desc,
	})

	// Sinyal 3: Telat kembali (6 bulan terakhir)
	var lateReturnsGroup, lateReturnsTotal int
	_ = DB.QueryRow(ctx, `
		SELECT COUNT(DISTINCT t.id)
		FROM public.borrow_transactions t
		JOIN public.borrow_items i ON t.id = i.transaction_id
		WHERE t.borrower_id = $1 AND t.group_name = $2
		  AND i.returned_at > i.due_date
		  AND i.returned_at >= NOW() - INTERVAL '6 months'
	`, trx.BorrowerID, trx.GroupName).Scan(&lateReturnsGroup)

	_ = DB.QueryRow(ctx, `
		SELECT COUNT(DISTINCT t.id)
		FROM public.borrow_transactions t
		JOIN public.borrow_items i ON t.id = i.transaction_id
		WHERE t.borrower_id = $1
		  AND i.returned_at > i.due_date
		  AND i.returned_at >= NOW() - INTERVAL '6 months'
	`, trx.BorrowerID).Scan(&lateReturnsTotal)

	s3Triggered := lateReturnsTotal >= 2
	s3Desc := fmt.Sprintf("Keterlambatan pengembalian: %d kali dalam 6 bulan.", lateReturnsTotal)
	if s3Triggered {
		s3Desc = fmt.Sprintf("Peminjam tercatat %d kali terlambat mengembalikan aset dalam 6 bulan terakhir (ambang batas >= 2).", lateReturnsTotal)
		triggeredSignals = append(triggeredSignals, s3Desc)
	}
	allSignals = append(allSignals, AnomalySignal{
		ID:          3,
		Name:        "Telat kembali (6 bulan terakhir)",
		Triggered:   s3Triggered,
		Description: s3Desc,
	})

	// Sinyal 4: Sering perpanjangan (6 bulan terakhir)
	var extGroup, extTotal int
	_ = DB.QueryRow(ctx, `
		SELECT COUNT(e.id)
		FROM public.borrow_extensions e
		JOIN public.borrow_transactions t ON e.transaction_id = t.id
		WHERE e.borrower_id = $1 AND t.group_name = $2
		  AND e.request_date >= NOW() - INTERVAL '6 months'
	`, trx.BorrowerID, trx.GroupName).Scan(&extGroup)

	_ = DB.QueryRow(ctx, `
		SELECT COUNT(id)
		FROM public.borrow_extensions
		WHERE borrower_id = $1 AND request_date >= NOW() - INTERVAL '6 months'
	`, trx.BorrowerID).Scan(&extTotal)

	s4Triggered := extTotal >= 2
	s4Desc := fmt.Sprintf("Pengajuan perpanjangan: %d kali dalam 6 bulan.", extTotal)
	if s4Triggered {
		s4Desc = fmt.Sprintf("Peminjam telah mengajukan perpanjangan durasi %d kali dalam 6 bulan terakhir.", extTotal)
		triggeredSignals = append(triggeredSignals, s4Desc)
	}
	allSignals = append(allSignals, AnomalySignal{
		ID:          4,
		Name:        "Sering perpanjangan",
		Triggered:   s4Triggered,
		Description: s4Desc,
	})

	// Sinyal 5: Jumlah aset per request (> 5 aset)
	s5Triggered := len(items) > 5
	s5Desc := fmt.Sprintf("Jumlah aset dalam permohonan: %d aset.", len(items))
	if s5Triggered {
		s5Desc = fmt.Sprintf("Jumlah aset yang diajukan (%d aset) melebihi batas kebiasaan transaksi (> 5 aset).", len(items))
		triggeredSignals = append(triggeredSignals, s5Desc)
	}
	allSignals = append(allSignals, AnomalySignal{
		ID:          5,
		Name:        "Jumlah aset per request vs kebiasaan",
		Triggered:   s5Triggered,
		Description: s5Desc,
	})

	// Sinyal 6: Menghindari jadwal Audit (rentang 7 hari sebelum jadwal audit mulai)
	s6Triggered := false
	s6Desc := "Tidak terdeteksi jadwal audit dalam rentang 7 hari ke depan."
	var auditCategory string
	var auditStartDate time.Time
	errAudit := DB.QueryRow(ctx, `
		SELECT category, TO_DATE(start, 'YYYY-MM-DD')
		FROM public.schedule
		WHERE group_name = $1
		  AND TO_DATE(start, 'YYYY-MM-DD') >= $2::date
		  AND TO_DATE(start, 'YYYY-MM-DD') <= ($2::date + INTERVAL '7 days')
		ORDER BY TO_DATE(start, 'YYYY-MM-DD') ASC
		LIMIT 1
	`, trx.GroupName, trx.RequestDate).Scan(&auditCategory, &auditStartDate)

	if errAudit == nil {
		s6Triggered = true
		diffDays := int(auditStartDate.Sub(trx.RequestDate).Hours() / 24)
		if diffDays < 0 {
			diffDays = 0
		}
		s6Desc = fmt.Sprintf("Permohonan diajukan mepet (%d hari sebelum) jadwal %s pada %s.", diffDays, auditCategory, auditStartDate.Format("02 Jan 2006"))
		triggeredSignals = append(triggeredSignals, s6Desc)
	}
	allSignals = append(allSignals, AnomalySignal{
		ID:          6,
		Name:        "Menghindari jadwal Audit",
		Triggered:   s6Triggered,
		Description: s6Desc,
	})

	// Sinyal 7: Pinjam ulang berturut-turut (aset sama, orang sama, interval <= 1 hari)
	s7Triggered := false
	s7Desc := "Tidak ada riwayat perputaran pinjam-kembalikan berulang jarak dekat."
	for _, it := range items {
		var cycleCount int
		_ = DB.QueryRow(ctx, `
			SELECT COUNT(DISTINCT t.id)
			FROM public.borrow_transactions t
			JOIN public.borrow_items i ON t.id = i.transaction_id
			WHERE t.borrower_id = $1 AND i.asset_id = $2
			  AND t.status = 'completed'
			  AND i.returned_at >= NOW() - INTERVAL '6 months'
		`, trx.BorrowerID, it.AssetID).Scan(&cycleCount)

		if cycleCount >= 2 {
			s7Triggered = true
			s7Desc = fmt.Sprintf("Aset '%s' memiliki siklus berulang pinjam-kembalikan-pinjam lagi oleh peminjam ini (%d kali dalam 6 bulan).", it.AssetName, cycleCount)
			triggeredSignals = append(triggeredSignals, s7Desc)
			break
		}
	}
	allSignals = append(allSignals, AnomalySignal{
		ID:          7,
		Name:        "Pinjam ulang berturut-turut",
		Triggered:   s7Triggered,
		Description: s7Desc,
	})

	// 4. Build Borrower Context Summary
	borrowerContext := []AnomalyContextItem{
		{
			Metric:   "Pinjaman aktif",
			GroupVal: activeLoansGroup,
			TotalVal: activeLoansTotal,
			Status:   statusLabel(s2Triggered),
		},
		{
			Metric:   "Telat kembali (6 bulan terakhir)",
			GroupVal: lateReturnsGroup,
			TotalVal: lateReturnsTotal,
			Status:   statusLabel(s3Triggered),
		},
		{
			Metric:   "Jumlah pengajuan perpanjangan (6 bulan terakhir)",
			GroupVal: extGroup,
			TotalVal: extTotal,
			Status:   statusLabel(s4Triggered),
		},
	}

	signalsCount := len(triggeredSignals)
	reviewRecommended := signalsCount >= 1

	// 5. Generate Narrative (AI Synthesizer or Empty/Error without placeholder fallback)
	narrative := ""
	errorMessage := ""
	isAI := false
	if signalsCount > 0 {
		log.Printf("[borrow_anomaly] Requesting AI narrative from eiai_go for %s (%d signals)...", trx.BorrowerName, signalsCount)
		aiNarrative, errAI := synthesizeNarrativeWithAI(trx.BorrowerName, trx.GroupName, triggeredSignals)
		if errAI != nil {
			log.Printf("[borrow_anomaly] AI narrative error: %v", errAI)
			errorMessage = errAI.Error()
		} else {
			log.Printf("[borrow_anomaly] Received AI narrative: %s", aiNarrative)
			narrative = aiNarrative
			isAI = true
		}
	}

	response := AnomalyReportResponse{
		DataAsOf:          time.Now().Format("02 Jan 2006 15:04:05 WIB"),
		TransactionCode:   trx.TransactionCode,
		BorrowerName:      trx.BorrowerName,
		GroupName:         trx.GroupName,
		BorrowerContext:   borrowerContext,
		RequestContext:    allSignals,
		SignalsCount:      signalsCount,
		ReviewRecommended: reviewRecommended,
		Narrative:         narrative,
		ErrorMessage:      errorMessage,
		IsAIGenerated:     isAI,
	}

	return c.JSON(response)
}

func statusLabel(triggered bool) string {
	if triggered {
		return "menonjol"
	}
	return "normal"
}

// synthesizeNarrativeWithAI calls eiai_go service to synthesize a flowing narrative
func synthesizeNarrativeWithAI(borrowerName, groupName string, signals []string) (string, error) {
	eiaiURL := os.Getenv("EIAI_URL")
	if eiaiURL == "" {
		eiaiURL = "http://127.0.0.1:8000"
	}
	baseURL := strings.TrimRight(eiaiURL, "/")
	if strings.HasSuffix(baseURL, "/rag") {
		baseURL = strings.TrimSuffix(baseURL, "/rag")
	}

	payload := map[string]interface{}{
		"borrower_name": borrowerName,
		"group_name":    groupName,
		"signals":       signals,
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", baseURL+"/anomaly/narrative", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 35 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("EIAI error %d: %s", resp.StatusCode, string(respBody))
	}

	var res struct {
		Narrative string `json:"narrative"`
		Error     string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if res.Error != "" {
		return "", fmt.Errorf("%s", res.Error)
	}

	return strings.TrimSpace(res.Narrative), nil
}

// ApproveBorrowTransaction handles POST /api/borrow/:id/approve
func ApproveBorrowTransaction(c *fiber.Ctx) error {
	idParam := c.Params("id")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	type ItemDecision struct {
		ID                   string  `json:"id"`
		ApprovedDurationDays *int    `json:"approved_duration_days"`
		Note                 *string `json:"note"`
		IsApproved           *bool   `json:"is_approved"`
	}

	var payload struct {
		Decision string         `json:"decision"` // "approve" or "reject"
		Items    []ItemDecision `json:"items"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	for _, itm := range payload.Items {
		itemStatus := "approved"
		if itm.IsApproved != nil && !*itm.IsApproved {
			itemStatus = "rejected"
		}
		_, _ = DB.Exec(ctx, `
			UPDATE public.borrow_items
			SET approved_duration_days = $1, note = $2, is_approved = $3, status = $4
			WHERE id::text = $5
		`, itm.ApprovedDurationDays, itm.Note, itm.IsApproved, itemStatus, itm.ID)
	}

	overallStatus := "approved"
	if payload.Decision == "reject" {
		overallStatus = "rejected"
	}

	_, err := DB.Exec(ctx, `
		UPDATE public.borrow_transactions
		SET status = $1
		WHERE id::text = $2 OR transaction_code = $2
	`, overallStatus, idParam)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Transaction approval processed successfully", "status": overallStatus})
}

// CreateBorrowRequest handles POST /api/borrow/request
func CreateBorrowRequest(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	type ReqItem struct {
		AssetID      string `json:"asset_id"`
		DurationDays int    `json:"duration_days"`
	}

	var payload struct {
		BorrowerName string    `json:"borrower_name"`
		GroupName    string    `json:"group_name"`
		ManagerName  string    `json:"manager_name"`
		Items        []ReqItem `json:"items"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if strings.TrimSpace(payload.BorrowerName) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Nama peminjam wajib diisi"})
	}
	if len(payload.Items) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Minimal 1 aset harus dipilih"})
	}

	if strings.TrimSpace(payload.GroupName) == "" {
		payload.GroupName = "qtera mandiri"
	}
	if strings.TrimSpace(payload.ManagerName) == "" {
		payload.ManagerName = "Direct Request"
	}

	// Generate transaction code TRX-BRW-XXX
	var nextNum int
	_ = DB.QueryRow(ctx, `
		SELECT COALESCE(MAX(SUBSTRING(transaction_code FROM 9)::int), 0) + 1
		FROM public.borrow_transactions
		WHERE transaction_code LIKE 'TRX-BRW-%'
	`).Scan(&nextNum)

	trxCode := fmt.Sprintf("TRX-BRW-%03d", nextNum)

	var trxID string
	err := DB.QueryRow(ctx, `
		INSERT INTO public.borrow_transactions (
			id, transaction_code, borrower_id, borrower_name, manager_name, group_name, request_date, status, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, CURRENT_DATE, 'pending_approval', NOW()
		) RETURNING id
	`, trxCode, strings.ToLower(strings.ReplaceAll(payload.BorrowerName, " ", "_")), payload.BorrowerName, payload.ManagerName, payload.GroupName).Scan(&trxID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal menyimpan transaksi peminjaman: " + err.Error()})
	}

	for _, it := range payload.Items {
		dur := it.DurationDays
		if dur <= 0 {
			dur = 14
		}
		_, errItem := DB.Exec(ctx, `
			INSERT INTO public.borrow_items (
				id, transaction_id, asset_id, duration_days, status
			) VALUES (
				gen_random_uuid(), $1, $2, $3, 'pending'
			)
		`, trxID, it.AssetID, dur)
		if errItem != nil {
			log.Printf("[CreateBorrowRequest] Error inserting item %s: %v", it.AssetID, errItem)
		}
	}

	return c.Status(201).JSON(fiber.Map{
		"message":          "Permohonan peminjaman berhasil diajukan",
		"transaction_id":   trxID,
		"transaction_code": trxCode,
	})
}
