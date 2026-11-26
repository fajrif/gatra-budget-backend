package handlers

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// KliringPotonganRequest is one deduction (potongan) the client took when paying.
type KliringPotonganRequest struct {
	DeductionTypeID string  `json:"deduction_type_id"`
	Jumlah          float64 `json:"jumlah"`
	Keterangan      *string `json:"keterangan"`
}

// KliringRequest clears an invoice: the payment date, the amount actually
// received and the deductions that explain the gap to Nilai Tagihan.
type KliringRequest struct {
	TglBayar        string                   `json:"tgl_bayar"`
	NilaiPenerimaan float64                  `json:"nilai_penerimaan"`
	Potongan        []KliringPotonganRequest `json:"potongan"`
}

// kliringPotongan is a validated KliringPotonganRequest.
type kliringPotongan struct {
	DeductionTypeID uuid.UUID
	Jumlah          float64
	Keterangan      *string
}

// kliringPlan is a validated KliringRequest, ready to be written.
type kliringPlan struct {
	TglBayar        time.Time
	NilaiPenerimaan float64
	Potongan        []kliringPotongan
}

// centsOf converts a rupiah amount to whole sen, so the balance check compares
// exact integers instead of floats.
func centsOf(v float64) int64 {
	return int64(math.Round(v * 100))
}

// formatRupiahGap renders a sen amount as Indonesian rupiah, e.g. "Rp 2.000.000,50".
func formatRupiahGap(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	digits := strconv.FormatInt(cents/100, 10)
	var grouped []byte
	for i, ch := range []byte(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			grouped = append(grouped, '.')
		}
		grouped = append(grouped, ch)
	}
	return fmt.Sprintf("%sRp %s,%02d", sign, grouped, cents%100)
}

// planKliring validates a KliringRequest against the billed amount and the
// billing date of the invoice it clears. It does not touch the database.
// The rule is nilai_penerimaan + SUM(potongan) = nilai, compared in sen.
func planKliring(nilai float64, tglTagihan time.Time, req *KliringRequest) (*kliringPlan, error) {
	if req.TglBayar == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "tgl_bayar wajib diisi")
	}
	tglBayar, err := helpers.ParseDateString(&req.TglBayar)
	if err != nil || tglBayar == nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "tgl_bayar tidak valid")
	}
	if !tglBayar.After(tglTagihan) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "tgl_bayar harus setelah tgl_tagihan")
	}
	if req.NilaiPenerimaan < 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "nilai_penerimaan tidak boleh negatif")
	}

	plan := &kliringPlan{
		TglBayar:        *tglBayar,
		NilaiPenerimaan: req.NilaiPenerimaan,
		Potongan:        make([]kliringPotongan, 0, len(req.Potongan)),
	}
	var potonganCents int64
	for i, p := range req.Potongan {
		if p.Jumlah <= 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("potongan_jumlah harus lebih dari 0 (baris %d)", i+1))
		}
		typeID, err := uuid.Parse(p.DeductionTypeID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("deduction_type_id tidak valid (baris %d)", i+1))
		}
		potonganCents += centsOf(p.Jumlah)
		plan.Potongan = append(plan.Potongan, kliringPotongan{
			DeductionTypeID: typeID,
			Jumlah:          p.Jumlah,
			Keterangan:      p.Keterangan,
		})
	}

	gap := centsOf(nilai) - centsOf(req.NilaiPenerimaan) - potonganCents
	switch {
	case gap > 0:
		return nil, fiber.NewError(fiber.StatusBadRequest,
			"nilai_penerimaan tidak seimbang dengan nilai tagihan: masih kurang "+formatRupiahGap(gap)+" yang belum dijelaskan potongan")
	case gap < 0:
		return nil, fiber.NewError(fiber.StatusBadRequest,
			"nilai_penerimaan + potongan melebihi nilai tagihan sebesar "+formatRupiahGap(-gap))
	}
	return plan, nil
}

// checkDeductionTypesExist returns an error when any referenced deduction type
// is missing from the master table.
func checkDeductionTypesExist(plan *kliringPlan) error {
	if len(plan.Potongan) == 0 {
		return nil
	}
	unique := make(map[uuid.UUID]bool, len(plan.Potongan))
	ids := make([]uuid.UUID, 0, len(plan.Potongan))
	for _, p := range plan.Potongan {
		if !unique[p.DeductionTypeID] {
			unique[p.DeductionTypeID] = true
			ids = append(ids, p.DeductionTypeID)
		}
	}
	var found int64
	if err := config.DB.Model(&models.DeductionType{}).Where("id IN ?", ids).Count(&found).Error; err != nil {
		return err
	}
	if int(found) != len(ids) {
		return fiber.NewError(fiber.StatusBadRequest, "jenis_potongan tidak ditemukan di master potongan")
	}
	return nil
}

// applyKliring marks the invoice as Kliring and replaces its deductions. The
// caller runs it inside a transaction.
func applyKliring(tx *gorm.DB, invoiceID uuid.UUID, plan *kliringPlan) error {
	if err := tx.Model(&models.Invoice{}).Where("id = ?", invoiceID).Updates(map[string]any{
		"status":           models.InvoiceStatusKliring,
		"tgl_bayar":        plan.TglBayar,
		"nilai_penerimaan": plan.NilaiPenerimaan,
		"updated_at":       time.Now(),
	}).Error; err != nil {
		return err
	}
	return replaceInvoiceDeductions(tx, invoiceID, plan.Potongan)
}

// replaceInvoiceDeductions deletes the invoice's deductions and writes the new
// ones. An empty list just removes them. The caller runs it inside a transaction.
func replaceInvoiceDeductions(tx *gorm.DB, invoiceID uuid.UUID, potongan []kliringPotongan) error {
	if err := tx.Where("invoice_id = ?", invoiceID).Delete(&models.InvoiceDeduction{}).Error; err != nil {
		return err
	}
	for _, p := range potongan {
		row := models.InvoiceDeduction{
			InvoiceID:       invoiceID,
			DeductionTypeID: p.DeductionTypeID,
			Jumlah:          p.Jumlah,
			Keterangan:      p.Keterangan,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// clearKliring moves an invoice back to Submit and removes its clearing data.
// The caller runs it inside a transaction.
func clearKliring(tx *gorm.DB, invoiceID uuid.UUID) error {
	if err := tx.Model(&models.Invoice{}).Where("id = ?", invoiceID).Updates(map[string]any{
		"status":           models.InvoiceStatusSubmit,
		"tgl_bayar":        nil,
		"nilai_penerimaan": nil,
		"updated_at":       time.Now(),
	}).Error; err != nil {
		return err
	}
	return replaceInvoiceDeductions(tx, invoiceID, nil)
}

// KliringInvoice godoc
// @Summary Clear an invoice (mark as paid) with its received amount and deductions
// @Description Sets status Kliring, tgl_bayar and nilai_penerimaan and replaces the deductions. Rejected unless nilai_penerimaan + SUM(potongan) equals the invoice nilai. Calling it again on a Kliring invoice edits the clearing.
// @Tags Invoices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Invoice ID (UUID)"
// @Param request body handlers.KliringRequest true "Payment date, received amount and deductions"
// @Success 200 {object} models.Invoice "Cleared invoice"
// @Failure 400 {object} map[string]any "Invalid body, or amounts do not balance"
// @Failure 404 {object} map[string]any "Invoice or deduction type not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/invoices/{id}/kliring [put]
func KliringInvoice(c *fiber.Ctx) error {
	var invoice models.Invoice
	if err := config.DB.Where("id = ?", c.Params("id")).First(&invoice).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invoice not found"})
	}

	var req KliringRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	plan, err := planKliring(invoice.Nilai, invoice.TglTagihan.Time, &req)
	if err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	if err := checkDeductionTypesExist(plan); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to clear invoice"})
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		return applyKliring(tx, invoice.ID, plan)
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to clear invoice"})
	}
	return c.JSON(loadInvoiceWithDeductions(invoice.ID))
}

// CancelKliringInvoice godoc
// @Summary Cancel the clearing of an invoice (back to Submit)
// @Description Clears tgl_bayar and nilai_penerimaan and deletes the deductions.
// @Tags Invoices
// @Security BearerAuth
// @Param id path string true "Invoice ID (UUID)"
// @Success 200 {object} models.Invoice "Invoice back to Submit"
// @Failure 400 {object} map[string]any "Invoice is not Kliring"
// @Failure 404 {object} map[string]any "Invoice not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/invoices/{id}/kliring [delete]
func CancelKliringInvoice(c *fiber.Ctx) error {
	var invoice models.Invoice
	if err := config.DB.Where("id = ?", c.Params("id")).First(&invoice).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invoice not found"})
	}
	if invoice.Status != models.InvoiceStatusKliring {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invoice belum Kliring"})
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		return clearKliring(tx, invoice.ID)
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to cancel clearing"})
	}
	return c.JSON(loadInvoiceWithDeductions(invoice.ID))
}

// loadInvoiceWithDeductions re-reads an invoice with the relations the frontend shows.
func loadInvoiceWithDeductions(id uuid.UUID) models.Invoice {
	var invoice models.Invoice
	config.DB.Preload("Project").Preload("Deductions.DeductionType").Where("id = ?", id).First(&invoice)
	return invoice
}
