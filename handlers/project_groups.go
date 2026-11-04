package handlers

import (
	"strings"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

// ProjectGroupResponse is a group plus the totals of the projects in it.
type ProjectGroupResponse struct {
	models.ProjectGroup
	ProjectCount int64   `gorm:"column:project_count" json:"project_count"`
	TotalNilai   float64 `gorm:"column:total_nilai" json:"total_nilai"`
}

// projectGroupSelect lists groups with the project count and the summed
// nilai_pekerjaan of their projects in one query.
func projectGroupSelect() string {
	return "project_groups.*, COUNT(projects.id) AS project_count, COALESCE(SUM(projects.nilai_pekerjaan), 0) AS total_nilai"
}

// projectGroupNameTaken reports whether another group already uses the name
// (case-insensitive). excludeID may be empty.
func projectGroupNameTaken(name, excludeID string) (bool, error) {
	var count int64
	q := config.DB.Model(&models.ProjectGroup{}).Where("LOWER(nama) = LOWER(?)", name)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ValidateProjectGroupRequest validates the group request fields.
func ValidateProjectGroupRequest(req *models.ProjectGroupRequest) error {
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nama is required")
	}
	return nil
}

// GetAllProjectGroups godoc
// @Summary Get all project groups
// @Description Groups with their project_count and total_nilai (sum of their projects' nilai_pekerjaan)
// @Tags ProjectGroups
// @Security BearerAuth
// @Param search query string false "Search by nama"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of project groups with pagination"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/project-groups [get]
func GetAllProjectGroups(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Model(&models.ProjectGroup{}).
		Select(projectGroupSelect()).
		Joins("LEFT JOIN projects ON projects.project_group_id = project_groups.id").
		Group("project_groups.id").
		Order("project_groups.nama ASC")
	queryCount := config.DB.Model(&models.ProjectGroup{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		query = query.Where("project_groups.nama ILIKE ?", term)
		queryCount = queryCount.Where("nama ILIKE ?", term)
	}

	var items []ProjectGroupResponse
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Scan(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch project groups"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, items, "project_groups", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}
	return c.JSON(response)
}

// GetProjectGroup godoc
// @Summary Get a project group by ID
// @Tags ProjectGroups
// @Security BearerAuth
// @Param id path string true "Project group ID (UUID)"
// @Success 200 {object} map[string]any "Project group details"
// @Failure 404 {object} map[string]any "Project group not found"
// @Router /api/project-groups/{id} [get]
func GetProjectGroup(c *fiber.Ctx) error {
	var item ProjectGroupResponse
	err := config.DB.Model(&models.ProjectGroup{}).
		Select(projectGroupSelect()).
		Joins("LEFT JOIN projects ON projects.project_group_id = project_groups.id").
		Where("project_groups.id = ?", c.Params("id")).
		Group("project_groups.id").
		Scan(&item).Error
	if err != nil || item.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project group not found"})
	}
	return c.JSON(fiber.Map{"project_group": item})
}

// CreateProjectGroup godoc
// @Summary Create a project group (admin only)
// @Tags ProjectGroups
// @Security BearerAuth
// @Param request body models.ProjectGroupRequest true "Project group details"
// @Success 201 {object} models.ProjectGroup "Created"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 409 {object} map[string]any "Name already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/project-groups [post]
func CreateProjectGroup(c *fiber.Ctx) error {
	var req models.ProjectGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := ValidateProjectGroupRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	taken, err := projectGroupNameTaken(req.Nama, "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create project group"})
	}
	if taken {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "nama already exists"})
	}

	item := models.ProjectGroup{Nama: req.Nama, Deskripsi: req.Deskripsi}
	if err := config.DB.Create(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create project group"})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

// UpdateProjectGroup godoc
// @Summary Update a project group (admin only)
// @Tags ProjectGroups
// @Security BearerAuth
// @Param id path string true "Project group ID (UUID)"
// @Param request body models.ProjectGroupRequest true "Updated details"
// @Success 200 {object} models.ProjectGroup "Updated"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 404 {object} map[string]any "Not found"
// @Failure 409 {object} map[string]any "Name already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/project-groups/{id} [put]
func UpdateProjectGroup(c *fiber.Ctx) error {
	var item models.ProjectGroup
	if err := config.DB.Where("id = ?", c.Params("id")).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project group not found"})
	}
	var req models.ProjectGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := ValidateProjectGroupRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	taken, err := projectGroupNameTaken(req.Nama, c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update project group"})
	}
	if taken {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "nama already exists"})
	}

	item.Nama = req.Nama
	item.Deskripsi = req.Deskripsi
	if err := config.DB.Save(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update project group"})
	}
	return c.JSON(item)
}

// DeleteProjectGroup godoc
// @Summary Delete a project group (admin only)
// @Description Rejected with 409 while projects still belong to it
// @Tags ProjectGroups
// @Security BearerAuth
// @Param id path string true "Project group ID (UUID)"
// @Success 200 {object} map[string]any "Deleted"
// @Failure 404 {object} map[string]any "Not found"
// @Failure 409 {object} map[string]any "Still in use"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/project-groups/{id} [delete]
func DeleteProjectGroup(c *fiber.Ctx) error {
	var used int64
	if err := config.DB.Model(&models.Project{}).Where("project_group_id = ?", c.Params("id")).Count(&used).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete project group"})
	}
	if used > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Project group is still used by projects"})
	}
	result := config.DB.Delete(&models.ProjectGroup{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete project group"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project group not found"})
	}
	return c.JSON(fiber.Map{"message": "Project group deleted successfully"})
}
