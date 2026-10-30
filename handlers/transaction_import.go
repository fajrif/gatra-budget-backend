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

// TransactionImportRow is one row of the transaction Excel template. The
// project is referenced by its No SP2K and the cost type by its name.
type TransactionImportRow struct {
	Row                int    `json:"row"`
	NoSP2K             string `json:"no_sp2k"`
	CostType           string `json:"cost_type"`
	TanggalTransaksi   string `json:"tanggal_transaksi"`
	JumlahRealisasi    string `json:"jumlah_realisasi"`
	DeskripsiRealisasi string `json:"deskripsi_realisasi"`
	JumlahTenagaKerja  string `json:"jumlah_tenaga_kerja"`
}

// ImportTransactions godoc
// @Summary Import transactions from Excel rows
// @Description Validates every row first; if any row is invalid nothing is saved and all errors are returned (422). Transactions have no natural key, so every row creates a new transaction.
// @Tags Transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "rows: list of transaction rows (see template)"
// @Success 200 {object} map[string]any "total, created, updated"
// @Failure 400 {object} map[string]any "Invalid body or too many rows"
// @Failure 422 {object} map[string]any "Validation errors per row"
// @Router /api/transactions/import [post]
func ImportTransactions(c *fiber.Ctx) error {
	rows, err := bindImportRows[TransactionImportRow](c)
	if err != nil {
		return respondImportError(c, err)
	}

	costTypes, err := loadNameIndex("cost_types", "nama_biaya")
	if err != nil {
		return respondImportError(c, err)
	}
	projects, err := loadProjectIndex()
	if err != nil {
		return respondImportError(c, err)
	}

	errs := &importErrors{}
	type prepared struct {
		req     *models.TransactionRequest
		project *models.Project
	}
	items := make([]prepared, len(rows))

	for i := range rows {
		r := &rows[i]
		r.NoSP2K = strings.TrimSpace(r.NoSP2K)
		r.CostType = strings.TrimSpace(r.CostType)
		r.TanggalTransaksi = strings.TrimSpace(r.TanggalTransaksi)
		r.JumlahRealisasi = strings.TrimSpace(r.JumlahRealisasi)
		r.JumlahTenagaKerja = strings.TrimSpace(r.JumlahTenagaKerja)

		before := len(errs.list)

		errs.requireFields(r.Row,
			importField{"no_sp2k", r.NoSP2K},
			importField{"cost_type", r.CostType},
			importField{"tanggal_transaksi", r.TanggalTransaksi},
			importField{"jumlah_realisasi", r.JumlahRealisasi},
		)

		req := &models.TransactionRequest{
			TanggalTransaksi:   r.TanggalTransaksi,
			DeskripsiRealisasi: optionalString(strings.TrimSpace(r.DeskripsiRealisasi)),
			// Placeholders keep the shared validator from reporting the lookups
			// below a second time; real IDs replace them on success.
			ProjectID:  uuid.Nil.String(),
			CostTypeID: uuid.Nil.String(),
		}

		var project *models.Project
		if r.NoSP2K != "" {
			if p, ok := projects[helpers.NormalizeKey(r.NoSP2K)]; ok {
				project = p
				req.ProjectID = p.ID.String()
			} else {
				errs.add(r.Row, "no_sp2k", "proyek dengan No SP2K '"+r.NoSP2K+"' tidak ditemukan")
			}
		}
		if r.CostType != "" {
			if id, ok := costTypes[helpers.NormalizeKey(r.CostType)]; ok {
				req.CostTypeID = id.String()
			} else {
				errs.add(r.Row, "cost_type", "jenis biaya '"+r.CostType+"' tidak ditemukan")
			}
		}

		if r.JumlahRealisasi != "" {
			if v, ok := helpers.ParseImportAmount(r.JumlahRealisasi); ok {
				req.JumlahRealisasi = v
			} else {
				errs.add(r.Row, "jumlah_realisasi", "harus berupa angka")
			}
		}
		if r.JumlahTenagaKerja != "" {
			if v, ok := helpers.ParseImportCount(r.JumlahTenagaKerja); ok {
				req.JumlahTenagaKerja = &v
			} else {
				errs.add(r.Row, "jumlah_tenaga_kerja", "harus berupa bilangan bulat tidak negatif")
			}
		}

		if len(errs.list) == before {
			if err := ValidateTransactionRequest(req); err != nil {
				errs.addValidation(r.Row, err)
			} else if err := applyTransactionRequest(&models.Transaction{}, req, project); err != nil {
				errs.addValidation(r.Row, err)
			}
		}
		items[i] = prepared{req: req, project: project}
	}

	if len(errs.list) > 0 {
		return respondImportInvalid(c, errs)
	}

	createdBy, _ := c.Locals("userEmail").(string)
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			transaction := models.Transaction{CreatedBy: createdBy}
			if err := applyTransactionRequest(&transaction, item.req, item.project); err != nil {
				return err
			}
			if err := tx.Create(&transaction).Error; err != nil {
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
