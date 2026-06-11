package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"boilerplates-backend/config"
	"boilerplates-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

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
// @Success 200 {object} map[string]interface{} "Upload successful with file URL"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation error"
// @Failure 404 {object} map[string]interface{} "Resource not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/upload/{resource}/{field}/{id} [post]
func UploadResourceField(c *fiber.Ctx) error {
	resource := c.Params("resource")
	field := c.Params("field")
	id := c.Params("id")

	validCombinations := map[string][]string{
		"users": {"photo"},
	}

	validFields, resourceExists := validCombinations[resource]
	if !resourceExists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid resource. Valid resources: users",
		})
	}

	fieldValid := false
	for _, vf := range validFields {
		if vf == field {
			fieldValid = true
			break
		}
	}
	if !fieldValid {
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

	fileExt := strings.ToLower(filepath.Ext(file.Filename))
	if fileExt != ".jpg" && fileExt != ".jpeg" && fileExt != ".png" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file type. Only JPEG and PNG images are allowed",
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

	if err := updateResourceFileURL(resource, field, id, fileURL); err != nil {
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

func updateResourceFileURL(resource, field, id, fileURL string) error {
	switch resource {
	case "users":
		var user models.User
		if err := config.DB.Where("id = ?", id).First(&user).Error; err != nil {
			return fmt.Errorf("resource not found")
		}
		return config.DB.Model(&user).Update("photo_url", fileURL).Error
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
// @Success 200 {object} map[string]interface{} "File deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request or validation error"
// @Failure 404 {object} map[string]interface{} "Resource or file not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/upload/{resource}/{field}/{id} [delete]
func DeleteResourceField(c *fiber.Ctx) error {
	resource := c.Params("resource")
	field := c.Params("field")
	id := c.Params("id")

	validCombinations := map[string][]string{
		"users": {"photo"},
	}

	validFields, resourceExists := validCombinations[resource]
	if !resourceExists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid resource. Valid resources: users",
		})
	}

	fieldValid := false
	for _, vf := range validFields {
		if vf == field {
			fieldValid = true
			break
		}
	}
	if !fieldValid {
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

	if err := updateResourceFileURL(resource, field, id, ""); err != nil {
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
	default:
		return nil, fmt.Errorf("invalid resource")
	}
}
