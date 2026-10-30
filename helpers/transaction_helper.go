package helpers

import "gatra-budget-backend/models"

// ComputeBulanRealisasi derives the "YYYY-MM" realization month from a
// transaction date. Monthly reports group on this value.
func ComputeBulanRealisasi(tanggalTransaksi models.Date) string {
	return tanggalTransaksi.Time.Format("2006-01")
}
