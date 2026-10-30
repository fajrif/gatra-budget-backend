package handlers

import (
	"strconv"
	"strings"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// InvoiceImportRow is one row of the invoice Excel template. The project is
// referenced by its No SP2K.
type InvoiceImportRow struct {
	Row          int    `json:"row"`
	NoSP2K       string `json:"no_sp2k"`
	NoInvoice    string `json:"no_invoice"`
	NamaInvoice  string `json:"nama_invoice"`
	TglTagihan   string `json:"tgl_tagihan"`
	TglBayar     string `json:"tgl_bayar"`
	Status       string `json:"status"`
	Nilai        string `json:"nilai"`
	ManajemenFee string `json:"manajemen_fee"`
	Description  string `json:"description"`
}

// canonicalInvoiceStatus lets the template accept "submit"/"CLEARING" in any
// case; unknown values are left as typed so the validator reports them.
func canonicalInvoiceStatus(s string) string {
	switch strings.ToLower(s) {
	case "submit":
		return models.InvoiceStatusSubmit
	case "clearing":
		return models.InvoiceStatusClearing
	}
	return s
}

// ImportInvoices godoc
// @Summary Import invoices from Excel rows
// @Description Validates every row first; if any row is invalid nothing is saved and all errors are returned (422). A row whose no_invoice already exists updates that invoice.
// @Tags Invoices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "rows: list of invoice rows (see template)"
// @Success 200 {object} map[string]any "total, created, updated"
// @Failure 400 {object} map[string]any "Invalid body or too many rows"
// @Failure 422 {object} map[string]any "Validation errors per row"
// @Router /api/invoices/import [post]
func ImportInvoices(c *fiber.Ctx) error {
	rows, err := bindImportRows[InvoiceImportRow](c)
	if err != nil {
		return respondImportError(c, err)
	}

	projects, err := loadProjectIndex()
	if err != nil {
		return respondImportError(c, err)
	}

	var existingList []models.Invoice
	if err := config.DB.Find(&existingList).Error; err != nil {
		return respondImportError(c, err)
	}
	existing := make(map[string]*models.Invoice, len(existingList))
	for i := range existingList {
		existing[existingList[i].NoInvoice] = &existingList[i]
	}

	errs := &importErrors{}
	seen := make(map[string]int, len(rows))
	requests := make([]*models.InvoiceRequest, len(rows))

	for i := range rows {
		r := &rows[i]
		r.NoSP2K = strings.TrimSpace(r.NoSP2K)
		r.NoInvoice = strings.TrimSpace(r.NoInvoice)
		r.NamaInvoice = strings.TrimSpace(r.NamaInvoice)
		r.TglTagihan = strings.TrimSpace(r.TglTagihan)
		r.TglBayar = strings.TrimSpace(r.TglBayar)
		r.Status = canonicalInvoiceStatus(strings.TrimSpace(r.Status))
		r.Nilai = strings.TrimSpace(r.Nilai)
		r.ManajemenFee = strings.TrimSpace(r.ManajemenFee)

		before := len(errs.list)

		errs.requireFields(r.Row,
			importField{"no_sp2k", r.NoSP2K},
			importField{"no_invoice", r.NoInvoice},
			importField{"nama_invoice", r.NamaInvoice},
			importField{"tgl_tagihan", r.TglTagihan},
			importField{"status", r.Status},
			importField{"nilai", r.Nilai},
		)

		if r.NoInvoice != "" {
			if first, dup := seen[r.NoInvoice]; dup {
				errs.add(r.Row, "no_invoice", "duplikat dengan baris "+strconv.Itoa(first)+" di file yang sama")
			} else {
				seen[r.NoInvoice] = r.Row
			}
		}

		req := &models.InvoiceRequest{
			NoInvoice:   r.NoInvoice,
			NamaInvoice: r.NamaInvoice,
			TglTagihan:  r.TglTagihan,
			TglBayar:    optionalString(r.TglBayar),
			Status:      r.Status,
			Description: optionalString(strings.TrimSpace(r.Description)),
			// Placeholder keeps the shared validator from reporting the project
			// lookup a second time; the real ID replaces it on success.
			ProjectID: "-",
		}

		if r.NoSP2K != "" {
			if p, ok := projects[helpers.NormalizeKey(r.NoSP2K)]; ok {
				req.ProjectID = p.ID.String()
			} else {
				errs.add(r.Row, "no_sp2k", "proyek dengan No SP2K '"+r.NoSP2K+"' tidak ditemukan")
			}
		}

		if r.Nilai != "" {
			if v, ok := helpers.ParseImportAmount(r.Nilai); ok {
				req.Nilai = v
			} else {
				errs.add(r.Row, "nilai", "harus berupa angka")
			}
		}
		if r.ManajemenFee != "" {
			if v, ok := helpers.ParseImportAmount(r.ManajemenFee); ok {
				req.ManajemenFee = &v
			} else {
				errs.add(r.Row, "manajemen_fee", "harus berupa angka")
			}
		}

		if len(errs.list) == before {
			if err := ValidateInvoiceRequest(req); err != nil {
				errs.addValidation(r.Row, err)
			} else if err := applyInvoiceRequest(&models.Invoice{}, req); err != nil {
				errs.addValidation(r.Row, err)
			}
		}
		requests[i] = req
	}

	if len(errs.list) > 0 {
		return respondImportInvalid(c, errs)
	}

	createdBy, _ := c.Locals("userEmail").(string)
	created, updated := 0, 0
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		for _, req := range requests {
			invoice, isUpdate := existing[req.NoInvoice]
			if !isUpdate {
				invoice = &models.Invoice{CreatedBy: createdBy}
			}
			if err := applyInvoiceRequest(invoice, req); err != nil {
				return err
			}
			if isUpdate {
				updated++
			} else {
				created++
			}
			if err := tx.Save(invoice).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return respondImportError(c, err)
	}
	return respondImportDone(c, len(rows), created, updated)
}
