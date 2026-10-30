package helpers

// AbsorptionAlertThreshold is the fraction of contract value realized above
// which a project is flagged for the absorption alert (80%).
const AbsorptionAlertThreshold = 0.8

// CalculateAbsorption returns the percentage of a project's contract value
// that has been realized, and whether that crosses AbsorptionAlertThreshold.
func CalculateAbsorption(nilaiPekerjaan, totalRealisasi float64) (pct float64, alert bool) {
	if nilaiPekerjaan == 0 {
		return 0, false
	}

	ratio := totalRealisasi / nilaiPekerjaan
	return ratio * 100, ratio > AbsorptionAlertThreshold
}

// CalculateRemainingBudget is the canonical project-level remaining budget:
// contract value minus total realized transactions.
func CalculateRemainingBudget(nilaiPekerjaan, totalRealisasi float64) float64 {
	return nilaiPekerjaan - totalRealisasi
}
