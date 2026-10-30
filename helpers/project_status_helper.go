package helpers

import "time"

// Contract statuses derived from a project's start and end dates.
const (
	ContractNotStarted = "not_started"
	ContractOngoing    = "ongoing"
	ContractExpired    = "expired"
)

// ContractStatus reports where today falls relative to the contract dates.
// Only the calendar date of each argument matters; the end date is the last
// day the contract is still ongoing.
func ContractStatus(mulai, selesai, today time.Time) string {
	mulai, selesai, today = dateOnly(mulai), dateOnly(selesai), dateOnly(today)

	switch {
	case today.Before(mulai):
		return ContractNotStarted
	case today.After(selesai):
		return ContractExpired
	default:
		return ContractOngoing
	}
}

// ContractEndingSoon is true for an ongoing contract whose end date is at most
// one calendar month away.
func ContractEndingSoon(mulai, selesai, today time.Time) bool {
	if ContractStatus(mulai, selesai, today) != ContractOngoing {
		return false
	}
	return !dateOnly(selesai).After(dateOnly(today).AddDate(0, 1, 0))
}

// ContractDaysRemaining is the number of days from today to the end date, or 0
// once the contract has ended.
func ContractDaysRemaining(selesai, today time.Time) int {
	days := int(dateOnly(selesai).Sub(dateOnly(today)).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
