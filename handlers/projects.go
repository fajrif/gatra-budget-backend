package handlers

import (
	"time"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

// ValidateProjectRequest validates the project request fields
func ValidateProjectRequest(req *models.ProjectRequest) error {
	if req.NoSP2K == "" {
		return fiber.NewError(fiber.StatusBadRequest, "no_sp2k is required")
	}
	if req.JudulPekerjaan == "" {
		return fiber.NewError(fiber.StatusBadRequest, "judul_pekerjaan is required")
	}
	if req.NilaiPekerjaan <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "nilai_pekerjaan is required")
	}
	if req.TanggalMulai == "" {
		return fiber.NewError(fiber.StatusBadRequest, "tanggal_mulai is required")
	}
	if req.TanggalSelesai == "" {
		return fiber.NewError(fiber.StatusBadRequest, "tanggal_selesai is required")
	}
	if req.TanggalSelesai <= req.TanggalMulai {
		return fiber.NewError(fiber.StatusBadRequest, "tanggal_selesai harus setelah tanggal_mulai")
	}
	if req.TanggalPerjanjian != nil && *req.TanggalPerjanjian != "" && *req.TanggalPerjanjian > req.TanggalMulai {
		return fiber.NewError(fiber.StatusBadRequest, "tanggal_perjanjian tidak boleh setelah tanggal_mulai")
	}
	return nil
}

// ProjectResponse wraps a Project with its contract status, which is derived
// from the dates on every read (not persisted): not_started, ongoing or
// expired, plus an ending-soon flag when an ongoing contract ends within a month.
type ProjectResponse struct {
	models.Project
	ContractStatus string `json:"contract_status"`
	EndingSoon     bool   `json:"ending_soon"`
	DaysRemaining  int    `json:"days_remaining"`
}

func toProjectResponse(p models.Project) ProjectResponse {
	today := time.Now()
	return ProjectResponse{
		Project:        p,
		ContractStatus: helpers.ContractStatus(p.TanggalMulai.Time, p.TanggalSelesai.Time, today),
		EndingSoon:     helpers.ContractEndingSoon(p.TanggalMulai.Time, p.TanggalSelesai.Time, today),
		DaysRemaining:  helpers.ContractDaysRemaining(p.TanggalSelesai.Time, today),
	}
}

// applyProjectRequest copies a validated ProjectRequest onto a Project,
// parsing dates and recomputing JangkaWaktu server-side. CreatedBy is only
// set on create (callers pass it empty for updates to leave it untouched).
func applyProjectRequest(p *models.Project, req *models.ProjectRequest) error {
	tanggalMulai, err := helpers.ParseDateString(&req.TanggalMulai)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tanggal_mulai: "+err.Error())
	}
	tanggalSelesai, err := helpers.ParseDateString(&req.TanggalSelesai)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tanggal_selesai: "+err.Error())
	}
	tanggalPerjanjian, err := helpers.ParseDateString(req.TanggalPerjanjian)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tanggal_perjanjian: "+err.Error())
	}

	p.NoSP2K = req.NoSP2K
	p.NoPerjanjian = req.NoPerjanjian
	p.NoAmandemen = req.NoAmandemen
	p.JudulPekerjaan = req.JudulPekerjaan
	p.NilaiPekerjaan = req.NilaiPekerjaan
	p.ClientID = helpers.ParseUUIDPtr(req.ClientID)
	p.ContractTypeID = helpers.ParseUUIDPtr(req.ContractTypeID)

	p.TanggalMulai = models.Date{Time: *tanggalMulai}
	p.TanggalSelesai = models.Date{Time: *tanggalSelesai}
	if tanggalPerjanjian != nil {
		p.TanggalPerjanjian = &models.Date{Time: *tanggalPerjanjian}
	} else {
		p.TanggalPerjanjian = nil
	}

	// JangkaWaktu is always server-computed, never bound from the request.
	months := helpers.CalculateDurationInMonths(*tanggalMulai, *tanggalSelesai)
	p.JangkaWaktu = &months

	return nil
}

// contractStatusCondition is the SQL form of helpers.ContractStatus, so list
// filtering agrees with the status shown on each row.
func contractStatusCondition(status string) (string, bool) {
	switch status {
	case helpers.ContractNotStarted:
		return "tanggal_mulai > CURRENT_DATE", true
	case helpers.ContractOngoing:
		return "tanggal_mulai <= CURRENT_DATE AND tanggal_selesai >= CURRENT_DATE", true
	case helpers.ContractExpired:
		return "tanggal_selesai < CURRENT_DATE", true
	}
	return "", false
}

// GetAllProjects godoc
// @Summary Get all projects
// @Description Retrieve all projects with optional status filter, search, and pagination
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param contract_status query string false "Filter by contract status (not_started, ongoing, expired)"
// @Param search query string false "Search by no_sp2k or judul_pekerjaan"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of projects with pagination"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/projects [get]
func GetAllProjects(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Preload("Client").Preload("ContractType").Order("created_at DESC")
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

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		cond := "no_sp2k ILIKE ? OR judul_pekerjaan ILIKE ?"
		query = query.Where(cond, term, term)
		queryCount = queryCount.Where(cond, term, term)
	}

	var projects []models.Project
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&projects).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch projects"})
	}

	responses := make([]ProjectResponse, len(projects))
	for i, p := range projects {
		responses[i] = toProjectResponse(p)
	}

	response, err := helpers.CreatePaginationResponse(queryCount, responses, "projects", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetProject godoc
// @Summary Get a project by ID
// @Tags Projects
// @Security BearerAuth
// @Param id path string true "Project ID (UUID)"
// @Success 200 {object} map[string]any "Project details"
// @Failure 404 {object} map[string]any "Project not found"
// @Router /api/projects/{id} [get]
func GetProject(c *fiber.Ctx) error {
	var project models.Project
	if err := config.DB.Preload("Client").Preload("ContractType").Where("id = ?", c.Params("id")).First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
	}
	return c.JSON(fiber.Map{"project": toProjectResponse(project)})
}

// CreateProject godoc
// @Summary Create a new project
// @Tags Projects
// @Security BearerAuth
// @Param request body models.ProjectRequest true "Project details"
// @Success 201 {object} models.Project "Created project"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 409 {object} map[string]any "no_sp2k already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/projects [post]
func CreateProject(c *fiber.Ctx) error {
	var req models.ProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := ValidateProjectRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	var existing models.Project
	if err := config.DB.Where("LOWER(no_sp2k) = LOWER(?)", req.NoSP2K).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "no_sp2k already exists"})
	}

	var project models.Project
	if err := applyProjectRequest(&project, &req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	if email, ok := c.Locals("userEmail").(string); ok {
		project.CreatedBy = email
	}

	if err := config.DB.Create(&project).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create project"})
	}
	return c.Status(fiber.StatusCreated).JSON(toProjectResponse(project))
}

// UpdateProject godoc
// @Summary Update a project
// @Tags Projects
// @Security BearerAuth
// @Param id path string true "Project ID (UUID)"
// @Param request body models.ProjectRequest true "Updated project details"
// @Success 200 {object} models.Project "Updated project"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 404 {object} map[string]any "Project not found"
// @Failure 409 {object} map[string]any "no_sp2k already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/projects/{id} [put]
func UpdateProject(c *fiber.Ctx) error {
	var project models.Project
	if err := config.DB.Where("id = ?", c.Params("id")).First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
	}

	var req models.ProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := ValidateProjectRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	var existing models.Project
	if err := config.DB.Where("LOWER(no_sp2k) = LOWER(?) AND id <> ?", req.NoSP2K, project.ID).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "no_sp2k already exists"})
	}

	if err := applyProjectRequest(&project, &req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	if err := config.DB.Save(&project).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update project"})
	}
	return c.JSON(toProjectResponse(project))
}

// DeleteProject godoc
// @Summary Delete a project
// @Tags Projects
// @Security BearerAuth
// @Param id path string true "Project ID (UUID)"
// @Success 200 {object} map[string]any "Project deleted successfully"
// @Failure 404 {object} map[string]any "Project not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/projects/{id} [delete]
func DeleteProject(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.Project{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete project"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
	}
	return c.JSON(fiber.Map{"message": "Project deleted successfully"})
}
