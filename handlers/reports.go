package handlers

import (
	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// These report endpoints are the only place remaining budget and profit margin
// leave the server. They sit behind middleware.AdminOnly() in routes.go; the
// plain /projects and /transactions payloads never include these figures.

// ProjectSummary is the admin-only financial view of one project.
type ProjectSummary struct {
	ProjectID                uuid.UUID `json:"project_id"`
	NoSP2K                   string    `json:"no_sp2k"`
	JudulPekerjaan           string    `json:"judul_pekerjaan"`
	StatusKontrak            string    `json:"status_kontrak"`
	NilaiPekerjaan           float64   `json:"nilai_pekerjaan"`
	TotalRealisasi           float64   `json:"total_realisasi"`
	RemainingBudget          float64   `json:"remaining_budget"`
	AbsorptionPct            float64   `json:"absorption_pct"`
	AbsorptionAlert          bool      `json:"absorption_alert"`
	TarifManagementFeePersen *float64  `json:"tarif_management_fee_persen"`
	ProfitMargin             float64   `json:"profit_margin"`
}

// ManagementFeeRow is one project's realized spend and fee for one month.
type ManagementFeeRow struct {
	ProjectID                uuid.UUID `json:"project_id"`
	NoSP2K                   string    `json:"no_sp2k"`
	JudulPekerjaan           string    `json:"judul_pekerjaan"`
	BulanRealisasi           string    `json:"bulan_realisasi"`
	TotalRealisasi           float64   `json:"total_realisasi"`
	TarifManagementFeePersen *float64  `json:"tarif_management_fee_persen"`
	ManagementFee            float64   `json:"management_fee"`
}

// ManagementFeeTotals sums the report. ManagementFee is computed per project
// from that project's full realized total, not by adding the monthly rows, so
// it matches ProjectSummary.ProfitMargin exactly.
type ManagementFeeTotals struct {
	TotalRealisasi float64 `json:"total_realisasi"`
	ManagementFee  float64 `json:"management_fee"`
}

// realisasiTotal is one GROUP BY result row from transactions.
type realisasiTotal struct {
	ProjectID uuid.UUID
	Total     float64
}

// monthlyRealisasiTotal is one GROUP BY result row from transactions, per month.
type monthlyRealisasiTotal struct {
	ProjectID      uuid.UUID
	BulanRealisasi string
	Total          float64
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

// buildProjectSummary applies the canonical project-level formulas to one
// project and its realized total.
func buildProjectSummary(p models.Project, totalRealisasi float64) ProjectSummary {
	absorptionPct, absorptionAlert := helpers.CalculateAbsorption(p.NilaiPekerjaan, totalRealisasi)

	return ProjectSummary{
		ProjectID:                p.ID,
		NoSP2K:                   p.NoSP2K,
		JudulPekerjaan:           p.JudulPekerjaan,
		StatusKontrak:            p.StatusKontrak,
		NilaiPekerjaan:           p.NilaiPekerjaan,
		TotalRealisasi:           totalRealisasi,
		RemainingBudget:          helpers.CalculateRemainingBudget(p.NilaiPekerjaan, totalRealisasi),
		AbsorptionPct:            absorptionPct,
		AbsorptionAlert:          absorptionAlert,
		TarifManagementFeePersen: p.TarifManagementFeePersen,
		ProfitMargin:             helpers.CalculateProfitMargin(p.TarifManagementFeePersen, totalRealisasi),
	}
}

// GetProjectsSummaryReport godoc
// @Summary Get remaining budget and profit margin for every project (admin only)
// @Description Per-project contract value, realized total, remaining budget, absorption (alert at 80%), and management-fee-style profit margin
// @Tags Reports
// @Security BearerAuth
// @Param status_kontrak query string false "Filter by status (Active, Pending, Non Aktif)"
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

	if status := c.Query("status_kontrak"); status != "" {
		query = query.Where("status_kontrak = ?", status)
		queryCount = queryCount.Where("status_kontrak = ?", status)
	}

	var projects []models.Project
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&projects).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch projects"})
	}

	projectIDs := make([]uuid.UUID, 0, len(projects))
	for _, p := range projects {
		projectIDs = append(projectIDs, p.ID)
	}
	totals, err := projectRealisasiTotals(config.DB, projectIDs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to aggregate transactions"})
	}

	summaries := make([]ProjectSummary, 0, len(projects))
	for _, p := range projects {
		summaries = append(summaries, buildProjectSummary(p, totals[p.ID]))
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

	totals, err := projectRealisasiTotals(config.DB, []uuid.UUID{project.ID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to aggregate transactions"})
	}

	return c.JSON(fiber.Map{"project_summary": buildProjectSummary(project, totals[project.ID])})
}

// GetManagementFeeReport godoc
// @Summary Get the management fee breakdown by project and month (admin only)
// @Description Realized spend and management fee per project per month, with grand totals. Computed server-side only; the client renders this as-is.
// @Tags Reports
// @Security BearerAuth
// @Success 200 {object} map[string]any "Rows by project and month, plus totals"
// @Failure 403 {object} map[string]any "Admin access required"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/reports/management-fee [get]
func GetManagementFeeReport(c *fiber.Ctx) error {
	var monthly []monthlyRealisasiTotal
	err := config.DB.Model(&models.Transaction{}).
		Select("project_id, bulan_realisasi, COALESCE(SUM(jumlah_realisasi), 0) AS total").
		Where("bulan_realisasi IS NOT NULL").
		Group("project_id, bulan_realisasi").
		Order("bulan_realisasi, project_id").
		Scan(&monthly).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to aggregate transactions"})
	}

	projectIDs := make([]uuid.UUID, 0)
	seen := make(map[uuid.UUID]bool)
	for _, m := range monthly {
		if !seen[m.ProjectID] {
			seen[m.ProjectID] = true
			projectIDs = append(projectIDs, m.ProjectID)
		}
	}

	projectsByID := make(map[uuid.UUID]models.Project, len(projectIDs))
	if len(projectIDs) > 0 {
		var projects []models.Project
		if err := config.DB.Where("id IN ?", projectIDs).Find(&projects).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch projects"})
		}
		for _, p := range projects {
			projectsByID[p.ID] = p
		}
	}

	// Full-project totals drive the grand totals so the fee is computed once
	// per project, not summed from rounded monthly figures.
	projectTotals := make(map[uuid.UUID]float64, len(projectIDs))
	rows := make([]ManagementFeeRow, 0, len(monthly))
	for _, m := range monthly {
		p, ok := projectsByID[m.ProjectID]
		if !ok {
			continue
		}
		projectTotals[m.ProjectID] += m.Total
		rows = append(rows, ManagementFeeRow{
			ProjectID:                p.ID,
			NoSP2K:                   p.NoSP2K,
			JudulPekerjaan:           p.JudulPekerjaan,
			BulanRealisasi:           m.BulanRealisasi,
			TotalRealisasi:           m.Total,
			TarifManagementFeePersen: p.TarifManagementFeePersen,
			ManagementFee:            helpers.CalculateProfitMargin(p.TarifManagementFeePersen, m.Total),
		})
	}

	var totals ManagementFeeTotals
	for id, total := range projectTotals {
		totals.TotalRealisasi += total
		totals.ManagementFee += helpers.CalculateProfitMargin(projectsByID[id].TarifManagementFeePersen, total)
	}

	return c.JSON(fiber.Map{"rows": rows, "totals": totals})
}
