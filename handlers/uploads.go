package handlers

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gatra-budget-backend/config"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func isValidField(validFields []string, field string) bool {
	return slices.Contains(validFields, field)
}

// UploadResourceField godoc
// @Summary Upload file for a resource
// @Description Upload image or file for a specific resource and field
// @Tags Upload
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param resource path string true "Resource name (users)"
// @Param field path string true "Field name (photo)"
// @Param id path string true "Resource ID (UUID)"
// @Param file formData file true "File to upload"
// @Success 200 {object} map[string]any "Upload successful with file URL"
// @Failure 400 {object} map[string]any "Invalid request or validation error"
// @Failure 404 {object} map[string]any "Resource not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/upload/{resource}/{field}/{id} [post]
func UploadResourceField(c *fiber.Ctx) error {
	resource := c.Params("resource")
	field := c.Params("field")
	id := c.Params("id")

	var validCombinations = map[string][]string{
		"users":        {"photo"},
		"transactions": {"bukti_transaksi"},
	}

	var validResourceNames = strings.Join(slices.Sorted(maps.Keys(validCombinations)), ", ")

	validFields, resourceExists := validCombinations[resource]
	if !resourceExists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid resource. Valid resources: " + validResourceNames,
		})
	}

	if !isValidField(validFields, field) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid field '%s' for resource '%s'. Valid fields: %s", field, resource, strings.Join(validFields, ", ")),
		})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	allowedExtensions := map[string][]string{
		"users/photo":                  {".jpg", ".jpeg", ".png"},
		"transactions/bukti_transaksi": {".jpg", ".jpeg", ".png", ".webp", ".pdf"},
	}
	fileExt := strings.ToLower(filepath.Ext(file.Filename))
	if !slices.Contains(allowedExtensions[resource+"/"+field], fileExt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file type for " + resource + "/" + field + ". Allowed: " + strings.Join(allowedExtensions[resource+"/"+field], ", "),
		})
	}
	if file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File size exceeds maximum allowed size of 5MB",
		})
	}

	uploadDir := filepath.Join("uploads", resource, field, id)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	timestamp := time.Now().Unix()
	uniqueFilename := fmt.Sprintf("%d_%s%s", timestamp, uuid.New().String()[:8], fileExt)
	filePath := filepath.Join(uploadDir, uniqueFilename)

	if err := c.SaveFile(file, filePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}

	fileURL := "/" + strings.ReplaceAll(filePath, "\\", "/")

	if err := updateResourceFileURL(resource, id, fileURL); err != nil {
		os.Remove(filePath)
		if err.Error() == "resource not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fmt.Sprintf("%s with ID %s not found", resource, id),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update database",
		})
	}

	return c.JSON(fiber.Map{
		"message":  "File uploaded successfully",
		"file_url": fileURL,
		"resource": resource,
		"field":    field,
		"id":       id,
	})
}

// updateResourceFileURL stores the file path on the given resource.
//
// The "field" path segment is a public API name, not a column name: the URL is
// /upload/users/photo/:id while the column is users.photo_url. Each resource
// therefore resolves its own column, mirroring getResourceFileURL.
func updateResourceFileURL(resource, id, fileURL string) error {
	switch resource {
	case "users":
		var user models.User
		if err := config.DB.Where("id = ?", id).First(&user).Error; err != nil {
			return fmt.Errorf("resource not found")
		}
		return config.DB.Model(&user).Update("photo_url", fileURL).Error
	case "transactions":
		var transaction models.Transaction
		if err := config.DB.Where("id = ?", id).First(&transaction).Error; err != nil {
			return fmt.Errorf("resource not found")
		}
		return config.DB.Model(&transaction).Update("bukti_transaksi_url", fileURL).Error
	default:
		return fmt.Errorf("invalid resource")
	}
}

// DeleteResourceField godoc
// @Summary Delete file for a resource
// @Description Delete uploaded file and set database field to null
// @Tags Upload
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param resource path string true "Resource name (users)"
// @Param field path string true "Field name (photo)"
// @Param id path string true "Resource ID (UUID)"
// @Success 200 {object} map[string]any "File deleted successfully"
// @Failure 400 {object} map[string]any "Invalid request or validation error"
// @Failure 404 {object} map[string]any "Resource or file not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/upload/{resource}/{field}/{id} [delete]
func DeleteResourceField(c *fiber.Ctx) error {
	resource := c.Params("resource")
	field := c.Params("field")
	id := c.Params("id")

	var validCombinations = map[string][]string{
		"users":        {"photo"},
		"transactions": {"bukti_transaksi"},
	}

	var validResourceNames = strings.Join(slices.Sorted(maps.Keys(validCombinations)), ", ")

	validFields, resourceExists := validCombinations[resource]
	if !resourceExists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid resource. Valid resources: " + validResourceNames,
		})
	}

	if !isValidField(validFields, field) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid field '%s' for resource '%s'. Valid fields: %s", field, resource, strings.Join(validFields, ", ")),
		})
	}

	currentFileURL, err := getResourceFileURL(resource, id)
	if err != nil {
		if err.Error() == "resource not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fmt.Sprintf("%s with ID %s not found", resource, id),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve resource",
		})
	}

	if currentFileURL == nil || *currentFileURL == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No file found for this resource field",
		})
	}

	filePath := strings.TrimPrefix(*currentFileURL, "/")
	if err := os.Remove(filePath); err != nil {
		fmt.Printf("Warning: Failed to delete file %s: %v\n", filePath, err)
	}

	if err := updateResourceFileURL(resource, id, ""); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update database",
		})
	}

	return c.JSON(fiber.Map{
		"message":  "File deleted successfully",
		"resource": resource,
		"field":    field,
		"id":       id,
	})
}

func getResourceFileURL(resource, id string) (*string, error) {
	switch resource {
	case "users":
		var user models.User
		if err := config.DB.Where("id = ?", id).First(&user).Error; err != nil {
			return nil, fmt.Errorf("resource not found")
		}
		return user.PhotoUrl, nil
	case "transactions":
		var transaction models.Transaction
		if err := config.DB.Where("id = ?", id).First(&transaction).Error; err != nil {
			return nil, fmt.Errorf("resource not found")
		}
		return transaction.BuktiTransaksiURL, nil
	default:
		return nil, fmt.Errorf("invalid resource")
	}
}
