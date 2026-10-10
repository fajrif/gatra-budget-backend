package handlers

import (
	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"
	"gatra-budget-backend/services"

	"github.com/gofiber/fiber/v2"
)

// ValidateBudgetItemRequest validates the budget item request fields
func ValidateBudgetItemRequest(req *models.BudgetItemRequest) error {
	if req.ProjectID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "project_id is required")
	}
	if req.CostTypeID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "cost_type_id is required")
	}
	if req.KategoriAnggaran != models.KategoriAnggaranMonthly && req.KategoriAnggaran != models.KategoriAnggaranLumpsum {
		return fiber.NewError(fiber.StatusBadRequest, "kategori_anggaran must be 'monthly' or 'lumpsum'")
	}
	if req.TotalAnggaran <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "total_anggaran is required and must be greater than 0")
	}
	return nil
}

// GetAllBudgetItems godoc
// @Summary Get all budget items
// @Description Retrieve budget items, optionally filtered by project or parent
// @Tags BudgetItems
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id query string false "Filter by project ID (UUID)"
// @Param parent_budget_id query string false "Filter by parent budget item ID (UUID)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of budget items with pagination"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/budget-items [get]
func GetAllBudgetItems(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Preload("CostType").Order("periode_bulan ASC, bulan_ke ASC")
	queryCount := config.DB.Model(&models.BudgetItem{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
		queryCount = queryCount.Where("project_id = ?", projectID)
	}

	if parentID := c.Query("parent_budget_id"); parentID != "" {
		query = query.Where("parent_budget_id = ?", parentID)
		queryCount = queryCount.Where("parent_budget_id = ?", parentID)
	}

	var budgetItems []models.BudgetItem
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&budgetItems).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch budget items"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, budgetItems, "budget_items", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetBudgetItem godoc
// @Summary Get a budget item by ID
// @Tags BudgetItems
// @Security BearerAuth
// @Param id path string true "Budget item ID (UUID)"
// @Success 200 {object} map[string]any "Budget item details"
// @Failure 404 {object} map[string]any "Budget item not found"
// @Router /api/budget-items/{id} [get]
func GetBudgetItem(c *fiber.Ctx) error {
	var budgetItem models.BudgetItem
	if err := config.DB.Preload("CostType").Where("id = ?", c.Params("id")).First(&budgetItem).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Budget item not found"})
	}
	return c.JSON(fiber.Map{"budget_item": budgetItem})
}

// CreateBudgetItem godoc
// @Summary Create a new budget item
// @Description Creates a parent budget item; "monthly" items also create evenly-split monthly children
// @Tags BudgetItems
// @Security BearerAuth
// @Param request body models.BudgetItemRequest true "Budget item details"
// @Success 201 {object} models.BudgetItem "Created budget item"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 404 {object} map[string]any "Project or cost type not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/budget-items [post]
func CreateBudgetItem(c *fiber.Ctx) error {
	var req models.BudgetItemRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := ValidateBudgetItemRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	var project models.Project
	if err := config.DB.Where("id = ?", req.ProjectID).First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
	}

	var costType models.CostType
	if err := config.DB.Where("id = ?", req.CostTypeID).First(&costType).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Cost type not found"})
	}

	splitter := services.NewBudgetItemSplitter(config.DB, &project)
	parent, err := splitter.Create(&costType, req.KategoriAnggaran, req.TotalAnggaran, req.DeskripsiAnggaran)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create budget item"})
	}

	return c.Status(fiber.StatusCreated).JSON(parent)
}

// UpdateBudgetItem godoc
// @Summary Update a budget item
// @Description Updates a parent budget item. Monthly items are re-synced: children are destroyed and rebuilt against the new total. Non-parent rows are updated directly.
// @Tags BudgetItems
// @Security BearerAuth
// @Param id path string true "Budget item ID (UUID)"
// @Param request body models.BudgetItemRequest true "Updated budget item details"
// @Success 200 {object} models.BudgetItem "Updated budget item"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 404 {object} map[string]any "Budget item not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/budget-items/{id} [put]
func UpdateBudgetItem(c *fiber.Ctx) error {
	var budgetItem models.BudgetItem
	if err := config.DB.Where("id = ?", c.Params("id")).First(&budgetItem).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Budget item not found"})
	}

	var req models.BudgetItemRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := ValidateBudgetItemRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	// Non-parent rows (monthly children) are plain CRUD — they don't own the
	// splitter lifecycle, only their parent does.
	if !budgetItem.IsParent {
		budgetItem.DeskripsiAnggaran = req.DeskripsiAnggaran
		if err := config.DB.Save(&budgetItem).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update budget item"})
		}
		return c.JSON(budgetItem)
	}

	var project models.Project
	if err := config.DB.Where("id = ?", budgetItem.ProjectID).First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
	}

	var costType *models.CostType
	if req.CostTypeID != "" && req.CostTypeID != budgetItem.CostTypeID.String() {
		var ct models.CostType
		if err := config.DB.Where("id = ?", req.CostTypeID).First(&ct).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Cost type not found"})
		}
		costType = &ct
	}

	// This handler deliberately branches instead of following the flat
	// parse->update->save template every other resource uses: a parent
	// budget item's children are owned by the splitter, not plain CRUD.
	splitter := services.NewBudgetItemSplitter(config.DB, &project)
	if err := splitter.Resync(&budgetItem, req.TotalAnggaran, req.DeskripsiAnggaran, costType); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update budget item"})
	}

	return c.JSON(budgetItem)
}

// DeleteBudgetItem godoc
// @Summary Delete a budget item
// @Description Deletes a budget item. Deleting a parent cascades to its monthly children.
// @Tags BudgetItems
// @Security BearerAuth
// @Param id path string true "Budget item ID (UUID)"
// @Success 200 {object} map[string]any "Budget item deleted successfully"
// @Failure 404 {object} map[string]any "Budget item not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/budget-items/{id} [delete]
func DeleteBudgetItem(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.BudgetItem{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete budget item"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Budget item not found"})
	}
	return c.JSON(fiber.Map{"message": "Budget item deleted successfully"})
}
