package handlers

import (
	"sort"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

// budgetTrendMonth is one row of the per-project budget-vs-actual chart.
// Not sensitive data (unlike contract value or margin), so this stays in the
// open AuthRequired() tier rather than behind AdminOnly().
type budgetTrendMonth struct {
	Month     string  `json:"month"`
	Anggaran  float64 `json:"anggaran"`
	Realisasi float64 `json:"realisasi"`
}

// GetProjectBudgetTrend godoc
// @Summary Get a project's monthly budget-vs-actual trend
// @Description Non-sensitive per-month allocated (budget item children) vs realized (transactions) totals
// @Tags Projects
// @Security BearerAuth
// @Param id path string true "Project ID (UUID)"
// @Success 200 {object} map[string]any "Monthly budget trend"
// @Router /api/projects/{id}/budget-trend [get]
func GetProjectBudgetTrend(c *fiber.Ctx) error {
	projectID := c.Params("id")

	var budgetChildren []models.BudgetItem
	if err := config.DB.Where("project_id = ? AND is_parent = false", projectID).Find(&budgetChildren).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch budget items"})
	}

	var transactions []models.Transaction
	if err := config.DB.Where("project_id = ?", projectID).Find(&transactions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch transactions"})
	}

	byMonth := map[string]*budgetTrendMonth{}
	monthRow := func(month string) *budgetTrendMonth {
		if row, ok := byMonth[month]; ok {
			return row
		}
		row := &budgetTrendMonth{Month: month}
		byMonth[month] = row
		return row
	}

	for _, b := range budgetChildren {
		if b.PeriodeBulan == nil {
			continue
		}
		monthRow(*b.PeriodeBulan).Anggaran += b.JumlahAnggaran
	}
	for _, t := range transactions {
		if t.BulanRealisasi == nil {
			continue
		}
		monthRow(*t.BulanRealisasi).Realisasi += t.JumlahRealisasi
	}

	trend := make([]budgetTrendMonth, 0, len(byMonth))
	for _, row := range byMonth {
		trend = append(trend, *row)
	}
	sort.Slice(trend, func(i, j int) bool { return trend[i].Month < trend[j].Month })

	return c.JSON(fiber.Map{"budget_trend": trend})
}

// GetProjectSLABreakdown godoc
// @Summary Get a project's payment SLA breakdown
// @Description Non-sensitive count of this project's transactions by SLA status
// @Tags Projects
// @Security BearerAuth
// @Param id path string true "Project ID (UUID)"
// @Success 200 {object} map[string]any "SLA breakdown"
// @Router /api/projects/{id}/sla-breakdown [get]
func GetProjectSLABreakdown(c *fiber.Ctx) error {
	var transactions []models.Transaction
	if err := config.DB.Where("project_id = ?", c.Params("id")).Find(&transactions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch transactions"})
	}

	breakdown := map[string]int{"on_time": 0, "h_plus_1": 0, "late": 0, "no_data": 0}
	for _, t := range transactions {
		breakdown[helpers.TransactionSLAStatus(t.TanggalTransaksi, t.TanggalPOTagihan)]++
	}

	return c.JSON(fiber.Map{"sla_breakdown": breakdown})
}
