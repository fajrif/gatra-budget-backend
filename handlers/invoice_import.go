package handlers

import (
	"strconv"
	"strings"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InvoiceImportRow is one row of the invoice Excel template. The project is
// referenced by its No SP2K.
type InvoiceImportRow struct {
	Row              int    `json:"row"`
	NoSP2K           string `json:"no_sp2k"`
	NoInvoice        string `json:"no_invoice"`
	NamaInvoice      string `json:"nama_invoice"`
	TglTagihan       string `json:"tgl_tagihan"`
	TglBayar         string `json:"tgl_bayar"`
	Status           string `json:"status"`
	MetodePembayaran string `json:"metode_pembayaran"`
	PeriodeTagihan   string `json:"periode_tagihan"`
	Nilai            string `json:"nilai"`
	NilaiPenerimaan  string `json:"nilai_penerimaan"`
	JenisPotongan    string `json:"jenis_potongan"`
	ManajemenFee     string `json:"manajemen_fee"`
	Description      string `json:"description"`
}

// canonicalInvoiceStatus lets the template accept "submit"/"KLIRING" in any
// case; unknown values are left as typed so the validator reports them.
func canonicalInvoiceStatus(s string) string {
	switch strings.ToLower(s) {
	case "submit":
		return models.InvoiceStatusSubmit
	case "kliring", "clearing": // "clearing" = old template spelling
		return models.InvoiceStatusKliring
	}
	return s
}

// canonicalPaymentMethod accepts "kontrak"/"SPR" in any case.
func canonicalPaymentMethod(s string) string {
	switch strings.ToLower(s) {
	case "kontrak":
		return models.PaymentMethodKontrak
	case "spr":
		return models.PaymentMethodSPR
	}
	return s
}

// splitPeriods splits the periode_tagihan cell ("2026-03, 2026-04") on commas,
// semicolons and whitespace.
func splitPeriods(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n'
	})
}

// buildImportKliring turns a Kliring row into a validated plan. The row's
// nilai_penerimaan defaults to nilai (nothing deducted). A lower value needs
// jenis_potongan, the master deduction type that takes the whole gap as one row.
// Several deductions per invoice are entered through the Kliring dialog instead.
func buildImportKliring(r *InvoiceImportRow, req *models.InvoiceRequest, typeByName map[string]uuid.UUID) (*kliringPlan, error) {
	tglTagihan, err := helpers.ParseDateString(&req.TglTagihan)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "tgl_tagihan tidak valid")
	}
	penerimaan := req.Nilai
	if r.NilaiPenerimaan != "" {
		v, ok := helpers.ParseImportAmount(r.NilaiPenerimaan)
		if !ok {
			return nil, fiber.NewError(fiber.StatusBadRequest, "nilai_penerimaan harus berupa angka")
		}
		penerimaan = v
	}

	var potongan []KliringPotonganRequest
	gap := centsOf(req.Nilai) - centsOf(penerimaan)
	switch {
	case gap > 0:
		if r.JenisPotongan == "" {
			return nil, fiber.NewError(fiber.StatusBadRequest, "jenis_potongan wajib diisi bila nilai_penerimaan lebih kecil dari nilai")
		}
		typeID, ok := typeByName[helpers.NormalizeKey(r.JenisPotongan)]
		if !ok {
			return nil, fiber.NewError(fiber.StatusBadRequest, "jenis_potongan '"+r.JenisPotongan+"' tidak ditemukan di master potongan")
		}
		potongan = []KliringPotonganRequest{{DeductionTypeID: typeID.String(), Jumlah: float64(gap) / 100}}
	case r.JenisPotongan != "":
		return nil, fiber.NewError(fiber.StatusBadRequest, "jenis_potongan hanya diisi bila nilai_penerimaan lebih kecil dari nilai")
	}

	return planKliring(req.Nilai, tglTagihan.UTC(), &KliringRequest{
		TglBayar:        r.TglBayar,
		NilaiPenerimaan: penerimaan,
		Potongan:        potongan,
	})
}

// ImportInvoices godoc
// @Summary Import invoices from Excel rows
// @Description Validates every row first; if any row is invalid nothing is saved and all errors are returned (422). A row whose no_invoice already exists updates that invoice. A Kliring row clears the invoice (tgl_bayar, nilai_penerimaan; a lower value needs jenis_potongan); a Submit row leaves it unpaid.
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

	var deductionTypes []models.DeductionType
	if err := config.DB.Find(&deductionTypes).Error; err != nil {
		return respondImportError(c, err)
	}
	typeByName := make(map[string]uuid.UUID, len(deductionTypes))
	for _, t := range deductionTypes {
		typeByName[helpers.NormalizeKey(t.Nama)] = t.ID
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
	plans := make([]*kliringPlan, len(rows))

	for i := range rows {
		r := &rows[i]
		r.NoSP2K = strings.TrimSpace(r.NoSP2K)
		r.NoInvoice = strings.TrimSpace(r.NoInvoice)
		r.NamaInvoice = strings.TrimSpace(r.NamaInvoice)
		r.TglTagihan = strings.TrimSpace(r.TglTagihan)
		r.TglBayar = strings.TrimSpace(r.TglBayar)
		r.Status = canonicalInvoiceStatus(strings.TrimSpace(r.Status))
		r.MetodePembayaran = canonicalPaymentMethod(strings.TrimSpace(r.MetodePembayaran))
		r.PeriodeTagihan = strings.TrimSpace(r.PeriodeTagihan)
		r.Nilai = strings.TrimSpace(r.Nilai)
		r.NilaiPenerimaan = strings.TrimSpace(r.NilaiPenerimaan)
		r.JenisPotongan = strings.TrimSpace(r.JenisPotongan)
		r.ManajemenFee = strings.TrimSpace(r.ManajemenFee)

		before := len(errs.list)

		errs.requireFields(r.Row,
			importField{"no_sp2k", r.NoSP2K},
			importField{"no_invoice", r.NoInvoice},
			importField{"nama_invoice", r.NamaInvoice},
			importField{"tgl_tagihan", r.TglTagihan},
			importField{"status", r.Status},
			importField{"metode_pembayaran", r.MetodePembayaran},
			importField{"periode_tagihan", r.PeriodeTagihan},
			importField{"nilai", r.Nilai},
		)

		switch r.Status {
		case models.InvoiceStatusSubmit:
			if r.TglBayar != "" {
				errs.add(r.Row, "tgl_bayar", "harus kosong bila status Submit")
			}
			if r.NilaiPenerimaan != "" || r.JenisPotongan != "" {
				errs.add(r.Row, "nilai_penerimaan", "hanya diisi bila status Kliring")
			}
		case models.InvoiceStatusKliring:
			// tgl_bayar, nilai_penerimaan and potongan are checked when the plan is built below.
		default:
			if r.Status != "" {
				errs.add(r.Row, "status", "harus Submit atau Kliring")
			}
		}

		if r.NoInvoice != "" {
			if first, dup := seen[r.NoInvoice]; dup {
				errs.add(r.Row, "no_invoice", "duplikat dengan baris "+strconv.Itoa(first)+" di file yang sama")
			} else {
				seen[r.NoInvoice] = r.Row
			}
		}

		req := &models.InvoiceRequest{
			NoInvoice:        r.NoInvoice,
			NamaInvoice:      r.NamaInvoice,
			TglTagihan:       r.TglTagihan,
			MetodePembayaran: r.MetodePembayaran,
			PeriodeTagihan:   splitPeriods(r.PeriodeTagihan),
			Description:      optionalString(strings.TrimSpace(r.Description)),
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

		var project *models.Project
		if r.NoSP2K != "" {
			project = projects[helpers.NormalizeKey(r.NoSP2K)]
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
			} else if err := ValidateInvoicePeriodInProject(req, project); err != nil {
				errs.addValidation(r.Row, err)
			} else if err := applyInvoiceRequest(&models.Invoice{}, req); err != nil {
				errs.addValidation(r.Row, err)
			}
		}
		if len(errs.list) == before && r.Status == models.InvoiceStatusKliring {
			plan, err := buildImportKliring(r, req, typeByName)
			if err != nil {
				errs.addValidation(r.Row, err)
			} else {
				plans[i] = plan
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
		for i, req := range requests {
			invoice, isUpdate := existing[req.NoInvoice]
			if !isUpdate {
				invoice = &models.Invoice{CreatedBy: createdBy}
			}
			if err := applyInvoiceRequest(invoice, req); err != nil {
				return err
			}

			// The clearing fields follow the row: Kliring sets them, Submit resets them.
			plan := plans[i]
			if plan != nil {
				paid := plan.NilaiPenerimaan
				invoice.Status = models.InvoiceStatusKliring
				invoice.TglBayar = &models.Date{Time: plan.TglBayar}
				invoice.NilaiPenerimaan = &paid
			} else {
				invoice.Status = models.InvoiceStatusSubmit
				invoice.TglBayar = nil
				invoice.NilaiPenerimaan = nil
			}

			if isUpdate {
				updated++
			} else {
				created++
			}
			if err := tx.Omit("Deductions").Save(invoice).Error; err != nil {
				return err
			}
			var potongan []kliringPotongan
			if plan != nil {
				potongan = plan.Potongan
			}
			if err := replaceInvoiceDeductions(tx, invoice.ID, potongan); err != nil {
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
