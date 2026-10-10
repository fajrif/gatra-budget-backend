package helpers

import (
	"math"
	"time"

	"gatra-budget-backend/models"
)

// ComputeBulanRealisasi derives the "YYYY-MM" realization month from a
// transaction date. Must match BudgetItem.PeriodeBulan's format exactly so
// monthly budget-vs-actual matching works.
func ComputeBulanRealisasi(tanggalTransaksi models.Date) string {
	return tanggalTransaksi.Time.Format("2006-01")
}

// ComputeManagementFee derives the management-fee rate and value for a
// transaction from the owning project's fee rate. Both return values are set
// together from the same input so they can never drift apart.
func ComputeManagementFee(jumlahRealisasi float64, tarifPersen *float64) (persentase, nilai *float64) {
	if tarifPersen == nil {
		return nil, nil
	}

	v := math.Round(jumlahRealisasi*(*tarifPersen/100)*100) / 100
	return tarifPersen, &v
}

// TransactionSLAStatus compares the realization date's month against the
// PO/invoice date's month: same month or earlier is on_time, exactly one
// month late is h_plus_1, anything later is late.
func TransactionSLAStatus(tanggalTransaksi models.Date, tanggalPOTagihan *models.Date) string {
	if tanggalPOTagihan == nil || tanggalPOTagihan.Time.IsZero() {
		return "no_data"
	}

	poMonth := monthIndex(tanggalPOTagihan.Time)
	txMonth := monthIndex(tanggalTransaksi.Time)

	switch {
	case txMonth <= poMonth:
		return "on_time"
	case txMonth == poMonth+1:
		return "h_plus_1"
	default:
		return "late"
	}
}

func monthIndex(t time.Time) int {
	return t.Year()*12 + int(t.Month())
}
