package handlers

import (
	"strings"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CostOperationImportRow is one row of the operational expense Excel template.
// The category is referenced by its name.
type CostOperationImportRow struct {
	Row              int    `json:"row"`
	CostType         string `json:"operational_cost_type"`
	TanggalTransaksi string `json:"tanggal_transaksi"`
	Jumlah           string `json:"jumlah"`
	Deskripsi        string `json:"deskripsi"`
}

// ImportCostOperations godoc
// @Summary Import operational expenses from Excel rows
// @Description Validates every row first; if any row is invalid nothing is saved and all errors are returned (422). Operational expenses have no natural key, so every row creates a new record.
// @Tags CostOperations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "rows: list of operational expense rows (see template)"
// @Success 200 {object} map[string]any "total, created, updated"
// @Failure 400 {object} map[string]any "Invalid body or too many rows"
// @Failure 422 {object} map[string]any "Validation errors per row"
// @Router /api/cost-operations/import [post]
func ImportCostOperations(c *fiber.Ctx) error {
	rows, err := bindImportRows[CostOperationImportRow](c)
	if err != nil {
		return respondImportError(c, err)
	}

	costTypes, err := loadNameIndex("operational_cost_types", "nama_biaya")
	if err != nil {
		return respondImportError(c, err)
	}

	errs := &importErrors{}
	items := make([]*models.CostOperationRequest, len(rows))

	for i := range rows {
		r := &rows[i]
		r.CostType = strings.TrimSpace(r.CostType)
		r.TanggalTransaksi = strings.TrimSpace(r.TanggalTransaksi)
		r.Jumlah = strings.TrimSpace(r.Jumlah)

		before := len(errs.list)

		errs.requireFields(r.Row,
			importField{"operational_cost_type", r.CostType},
			importField{"tanggal_transaksi", r.TanggalTransaksi},
			importField{"jumlah", r.Jumlah},
		)

		req := &models.CostOperationRequest{
			TanggalTransaksi: r.TanggalTransaksi,
			Deskripsi:        optionalString(strings.TrimSpace(r.Deskripsi)),
			// Placeholder keeps the shared validator from reporting the lookup
			// below a second time; the real ID replaces it on success.
			OperationalCostTypeID: uuid.Nil.String(),
		}

		if r.CostType != "" {
			if id, ok := costTypes[helpers.NormalizeKey(r.CostType)]; ok {
				req.OperationalCostTypeID = id.String()
			} else {
				errs.add(r.Row, "operational_cost_type", "jenis beban operasional '"+r.CostType+"' tidak ditemukan")
			}
		}

		if r.Jumlah != "" {
			if v, ok := helpers.ParseImportAmount(r.Jumlah); ok {
				req.Jumlah = v
			} else {
				errs.add(r.Row, "jumlah", "harus berupa angka")
			}
		}

		if len(errs.list) == before {
			if err := ValidateCostOperationRequest(req); err != nil {
				errs.addValidation(r.Row, err)
			} else if err := applyCostOperationRequest(&models.CostOperation{}, req); err != nil {
				errs.addValidation(r.Row, err)
			}
		}
		items[i] = req
	}

	if len(errs.list) > 0 {
		return respondImportInvalid(c, errs)
	}

	createdBy, _ := c.Locals("userEmail").(string)
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		for _, req := range items {
			item := models.CostOperation{CreatedBy: createdBy}
			if err := applyCostOperationRequest(&item, req); err != nil {
				return err
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return respondImportError(c, err)
	}
	return respondImportDone(c, len(rows), len(rows), 0)
}
