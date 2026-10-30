package handlers

import (
	"fmt"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// importErrors collects every problem found in an import so the user can fix
// them all in one pass.
type importErrors struct {
	list []helpers.ImportRowError
}

func (e *importErrors) add(row int, field, message string) {
	e.list = append(e.list, helpers.ImportRowError{Row: row, Field: field, Message: message})
}

// addValidation records an error returned by a Validate*/apply* function.
func (e *importErrors) addValidation(row int, err error) {
	field, message := helpers.TranslateValidationMessage(err.Error())
	e.add(row, field, message)
}

// importField is a required import column and the value the row has for it.
type importField struct {
	name  string
	value string
}

// requireFields reports every empty required field of a row and returns true
// when none is missing.
func (e *importErrors) requireFields(row int, fields ...importField) bool {
	ok := true
	for _, f := range fields {
		if f.value == "" {
			e.add(row, f.name, "wajib diisi")
			ok = false
		}
	}
	return ok
}

// bindImportRows reads the {"rows": [...]} body, enforcing the row limits.
func bindImportRows[T any](c *fiber.Ctx) ([]T, error) {
	var body struct {
		Rows []T `json:"rows"`
	}
	if err := c.BodyParser(&body); err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if len(body.Rows) == 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "File tidak berisi baris data")
	}
	if len(body.Rows) > helpers.MaxImportRows {
		return nil, fiber.NewError(fiber.StatusBadRequest,
			fmt.Sprintf("Maksimal %d baris per import", helpers.MaxImportRows))
	}
	return body.Rows, nil
}

func respondImportError(c *fiber.Ctx, err error) error {
	if fe, ok := err.(*fiber.Error); ok {
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Import gagal"})
}

// respondImportInvalid is the 422 returned when any row failed validation;
// nothing has been saved at that point.
func respondImportInvalid(c *fiber.Ctx, errs *importErrors) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
		"error":  "Validasi import gagal, tidak ada data yang disimpan",
		"errors": errs.list,
	})
}

func respondImportDone(c *fiber.Ctx, total, created, updated int) error {
	return c.JSON(fiber.Map{"total": total, "created": created, "updated": updated})
}

// loadNameIndex maps the normalized values of a master-data table's name
// column to their IDs.
func loadNameIndex(table, column string) (map[string]uuid.UUID, error) {
	var rows []struct {
		ID   uuid.UUID
		Name string
	}
	if err := config.DB.Table(table).Select("id, " + column + " AS name").Scan(&rows).Error; err != nil {
		return nil, err
	}
	index := make(map[string]uuid.UUID, len(rows))
	for _, r := range rows {
		index[helpers.NormalizeKey(r.Name)] = r.ID
	}
	return index, nil
}

// loadProjectIndex maps normalized No SP2K to the project.
func loadProjectIndex() (map[string]*models.Project, error) {
	var list []models.Project
	if err := config.DB.Find(&list).Error; err != nil {
		return nil, err
	}
	index := make(map[string]*models.Project, len(list))
	for i := range list {
		index[helpers.NormalizeKey(list[i].NoSP2K)] = &list[i]
	}
	return index, nil
}
