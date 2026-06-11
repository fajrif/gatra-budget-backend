package helpers

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type PaginationParams struct {
	Page   int
	Limit  int
	Offset int
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// GetPaginationParams extracts and validates pagination parameters from the request
func GetPaginationParams(c *fiber.Ctx) PaginationParams {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	offset := (page - 1) * limit

	return PaginationParams{
		Page:   page,
		Limit:  limit,
		Offset: offset,
	}
}

// CreatePaginationResponse creates a standardized pagination response with total count
// queryCount should be a separate query object using config.DB.Model(&YourModel{})
func CreatePaginationResponse(queryCount *gorm.DB, data interface{}, key string, page int, limit int) (fiber.Map, error) {
	var total int64

	// Count total records (without pagination)
	if err := queryCount.Count(&total).Error; err != nil {
		return nil, err
	}

	// Calculate total pages (ceiling division)
	totalPages := 0
	if total > 0 && limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return fiber.Map{
		key: data,
		"pagination": PaginationMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}
