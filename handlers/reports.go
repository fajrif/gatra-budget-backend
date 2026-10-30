package handlers

import (
	"strconv"
	"time"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// These report endpoints are the only place remaining budget and profit margin
// leave the server. They sit behind middleware.AdminOnly() in routes.go; the
// plain /projects, /transactions and /invoices payloads never include these
// aggregates.

// ProjectSummary is the admin-only financial view of one project. Profit
// margin is the sum of the fixed manajemen_fee across the project's invoices.
type ProjectSummary struct {
	ProjectID         uuid.UUID `json:"project_id"`
	NoSP2K            string    `json:"no_sp2k"`
	JudulPekerjaan    string    `json:"judul_pekerjaan"`
	ContractStatus    string    `json:"contract_status"`
	NilaiPekerjaan    float64   `json:"nilai_pekerjaan"`
	TotalRealisasi    float64   `json:"total_realisasi"`
	RemainingBudget   float64   `json:"remaining_budget"`
	AbsorptionPct     float64   `json:"absorption_pct"`
	AbsorptionAlert   bool      `json:"absorption_alert"`
	TotalInvoice      float64   `json:"total_invoice"`
	TotalClearing     float64   `json:"total_clearing"`
	TotalOutstanding  float64   `json:"total_outstanding"`
	TotalManajemenFee float64   `json:"total_manajemen_fee"`
	ProfitMargin      float64   `json:"profit_margin"`
}

// MonthlyReportRow is one calendar month of the monthly report.
type MonthlyReportRow struct {
	Month        string  `json:"month"`
	Pengeluaran  float64 `json:"pengeluaran"`
	Tagihan      float64 `json:"tagihan"`
	Penerimaan   float64 `json:"penerimaan"`
	ManajemenFee float64 `json:"manajemen_fee"`
	Selisih      float64 `json:"selisih"`
}

// MonthlyReportTotals sums the twelve months of the report.
type MonthlyReportTotals struct {
	Pengeluaran  float64 `json:"pengeluaran"`
	Tagihan      float64 `json:"tagihan"`
	Penerimaan   float64 `json:"penerimaan"`
	ManajemenFee float64 `json:"manajemen_fee"`
	Selisih      float64 `json:"selisih"`
}

// realisasiTotal is one GROUP BY result row from transactions.
type realisasiTotal struct {
	ProjectID uuid.UUID
	Total     float64
}

// invoiceTotal is one GROUP BY result row from invoices, per project.
type invoiceTotal struct {
	ProjectID        uuid.UUID
	TotalInvoice     float64
	TotalClearing    float64
	TotalOutstanding float64
	TotalFee         float64
}

// monthTotal is one GROUP BY result row keyed by "YYYY-MM".
type monthTotal struct {
	Month string
	Total float64
}

// projectRealisasiTotals returns SUM(jumlah_realisasi) keyed by project for the
// given projects. Projects with no transactions are absent from the map, which
// callers read as zero.
func projectRealisasiTotals(db *gorm.DB, projectIDs []uuid.UUID) (map[uuid.UUID]float64, error) {
	totals := make(map[uuid.UUID]float64, len(projectIDs))
	if len(projectIDs) == 0 {
		return totals, nil
	}

	var rows []realisasiTotal
	err := db.Model(&models.Transaction{}).
		Select("project_id, COALESCE(SUM(jumlah_realisasi), 0) AS total").
		Where("project_id IN ?", projectIDs).
		Group("project_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		totals[row.ProjectID] = row.Total
	}
	return totals, nil
}

// projectInvoiceTotals aggregates each project's invoices in one grouped query.
// Projects without invoices are absent from the map, which callers read as zero.
func projectInvoiceTotals(db *gorm.DB, projectIDs []uuid.UUID) (map[uuid.UUID]invoiceTotal, error) {
	totals := make(map[uuid.UUID]invoiceTotal, len(projectIDs))
	if len(projectIDs) == 0 {
		return totals, nil
	}

	var rows []invoiceTotal
	err := db.Model(&models.Invoice{}).
		Select(`project_id,
			COALESCE(SUM(nilai), 0) AS total_invoice,
			COALESCE(SUM(nilai) FILTER (WHERE status = 'Clearing'), 0) AS total_clearing,
			COALESCE(SUM(nilai) FILTER (WHERE status = 'Submit'), 0) AS total_outstanding,
			COALESCE(SUM(manajemen_fee), 0) AS total_fee`).
		Where("project_id IN ?", projectIDs).
		Group("project_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		totals[row.ProjectID] = row
	}
	return totals, nil
}

// buildProjectSummary applies the canonical project-level formulas to one
// project, its realized total and its invoice totals.
func buildProjectSummary(p models.Project, totalRealisasi float64, inv invoiceTotal) ProjectSummary {
	absorptionPct, absorptionAlert := helpers.CalculateAbsorption(p.NilaiPekerjaan, totalRealisasi)

	return ProjectSummary{
		ProjectID:         p.ID,
		NoSP2K:            p.NoSP2K,
		JudulPekerjaan:    p.JudulPekerjaan,
		ContractStatus:    helpers.ContractStatus(p.TanggalMulai.Time, p.TanggalSelesai.Time, time.Now()),
		NilaiPekerjaan:    p.NilaiPekerjaan,
		TotalRealisasi:    totalRealisasi,
		RemainingBudget:   helpers.CalculateRemainingBudget(p.NilaiPekerjaan, totalRealisasi),
		AbsorptionPct:     absorptionPct,
		AbsorptionAlert:   absorptionAlert,
		TotalInvoice:      inv.TotalInvoice,
		TotalClearing:     inv.TotalClearing,
		TotalOutstanding:  inv.TotalOutstanding,
		TotalManajemenFee: inv.TotalFee,
		ProfitMargin:      inv.TotalFee,
	}
}

// GetProjectsSummaryReport godoc
// @Summary Get remaining budget and profit margin for every project (admin only)
// @Description Per-project contract value, realized total, remaining budget, absorption (alert at 80%), invoice totals, and profit margin (sum of invoice manajemen_fee)
// @Tags Reports
// @Security BearerAuth
// @Param contract_status query string false "Filter by contract status (not_started, ongoing, expired)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "Project summaries with pagination"
// @Failure 403 {object} map[string]any "Admin access required"
// @Router /api/reports/projects-summary [get]
func GetProjectsSummaryReport(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Order("created_at DESC")
	queryCount := config.DB.Model(&models.Project{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if status := c.Query("contract_status"); status != "" {
		cond, ok := contractStatusCondition(status)
		if !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "contract_status must be not_started, ongoing, or expired"})
		}
		query = query.Where(cond)
		queryCount = queryCount.Where(cond)
	}

	var projects []models.Project
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&projects).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch projects"})
	}

	projectIDs := make([]uuid.UUID, 0, len(projects))
	for _, p := range projects {
		projectIDs = append(projectIDs, p.ID)
	}
	realisasi, err := projectRealisasiTotals(config.DB, projectIDs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to aggregate transactions"})
	}
	invoices, err := projectInvoiceTotals(config.DB, projectIDs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to aggregate invoices"})
	}

	summaries := make([]ProjectSummary, 0, len(projects))
	for _, p := range projects {
		summaries = append(summaries, buildProjectSummary(p, realisasi[p.ID], invoices[p.ID]))
	}

	response, err := helpers.CreatePaginationResponse(queryCount, summaries, "projects_summary", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetProjectSummaryReport godoc
// @Summary Get remaining budget and profit margin for one project (admin only)
// @Tags Reports
// @Security BearerAuth
// @Param id path string true "Project ID (UUID)"
// @Success 200 {object} map[string]any "Project summary"
// @Failure 403 {object} map[string]any "Admin access required"
// @Failure 404 {object} map[string]any "Project not found"
// @Router /api/reports/projects/{id}/summary [get]
func GetProjectSummaryReport(c *fiber.Ctx) error {
	var project models.Project
	if err := config.DB.Where("id = ?", c.Params("id")).First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
	}

	realisasi, err := projectRealisasiTotals(config.DB, []uuid.UUID{project.ID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to aggregate transactions"})
	}
	invoices, err := projectInvoiceTotals(config.DB, []uuid.UUID{project.ID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to aggregate invoices"})
	}

	return c.JSON(fiber.Map{"project_summary": buildProjectSummary(project, realisasi[project.ID], invoices[project.ID])})
}

// monthTotals runs one grouped query and returns totals keyed by "YYYY-MM".
func monthTotals(q *gorm.DB) (map[string]float64, error) {
	var rows []monthTotal
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make(map[string]float64, len(rows))
	for _, r := range rows {
		out[r.Month] = r.Total
	}
	return out, nil
}

// GetMonthlyReport godoc
// @Summary Get the monthly expense and invoice report (admin only)
// @Description Twelve months of one year: expenses (transactions by bulan_realisasi), billed invoices (by tgl_tagihan), received payments and fees (Clearing invoices by tgl_bayar), and received minus expenses. Optionally scoped to one project.
// @Tags Reports
// @Security BearerAuth
// @Param year query int false "Year (default: current year)"
// @Param project_id query string false "Limit to one project (UUID)"
// @Success 200 {object} map[string]any "Rows per month, plus totals"
// @Failure 400 {object} map[string]any "Invalid year"
// @Failure 403 {object} map[string]any "Admin access required"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/reports/monthly [get]
func GetMonthlyReport(c *fiber.Ctx) error {
	year := time.Now().Year()
	if y := c.Query("year"); y != "" {
		parsed, err := strconv.Atoi(y)
		if err != nil || parsed < 2000 || parsed > 2100 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "year must be between 2000 and 2100"})
		}
		year = parsed
	}
	projectID := c.Query("project_id")
	prefix := strconv.Itoa(year) + "-%"

	scope := func(db *gorm.DB) *gorm.DB {
		if projectID != "" {
			return db.Where("project_id = ?", projectID)
		}
		return db
	}

	pengeluaran, err := monthTotals(scope(config.DB.Model(&models.Transaction{})).
		Select("bulan_realisasi AS month, COALESCE(SUM(jumlah_realisasi), 0) AS total").
		Where("bulan_realisasi LIKE ?", prefix).
		Group("bulan_realisasi"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to aggregate transactions"})
	}

	tagihan, err := monthTotals(scope(config.DB.Model(&models.Invoice{})).
		Select("to_char(tgl_tagihan, 'YYYY-MM') AS month, COALESCE(SUM(nilai), 0) AS total").
		Where("to_char(tgl_tagihan, 'YYYY-MM') LIKE ?", prefix).
		Group("to_char(tgl_tagihan, 'YYYY-MM')"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to aggregate invoices"})
	}

	penerimaan, err := monthTotals(scope(config.DB.Model(&models.Invoice{})).
		Select("to_char(tgl_bayar, 'YYYY-MM') AS month, COALESCE(SUM(nilai), 0) AS total").
		Where("status = ? AND to_char(tgl_bayar, 'YYYY-MM') LIKE ?", models.InvoiceStatusClearing, prefix).
		Group("to_char(tgl_bayar, 'YYYY-MM')"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to aggregate invoices"})
	}

	fee, err := monthTotals(scope(config.DB.Model(&models.Invoice{})).
		Select("to_char(tgl_bayar, 'YYYY-MM') AS month, COALESCE(SUM(manajemen_fee), 0) AS total").
		Where("status = ? AND to_char(tgl_bayar, 'YYYY-MM') LIKE ?", models.InvoiceStatusClearing, prefix).
		Group("to_char(tgl_bayar, 'YYYY-MM')"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to aggregate invoices"})
	}

	rows := make([]MonthlyReportRow, 0, 12)
	var totals MonthlyReportTotals
	for m := 1; m <= 12; m++ {
		key := time.Date(year, time.Month(m), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
		row := MonthlyReportRow{
			Month:        key,
			Pengeluaran:  pengeluaran[key],
			Tagihan:      tagihan[key],
			Penerimaan:   penerimaan[key],
			ManajemenFee: fee[key],
			Selisih:      penerimaan[key] - pengeluaran[key],
		}
		rows = append(rows, row)

		totals.Pengeluaran += row.Pengeluaran
		totals.Tagihan += row.Tagihan
		totals.Penerimaan += row.Penerimaan
		totals.ManajemenFee += row.ManajemenFee
		totals.Selisih += row.Selisih
	}

	return c.JSON(fiber.Map{"year": year, "rows": rows, "totals": totals})
}
