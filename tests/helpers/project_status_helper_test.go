package helpers_test

import (
	"testing"
	"time"

	"gatra-budget-backend/helpers"

	"github.com/stretchr/testify/assert"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestContractStatus(t *testing.T) {
	mulai, selesai := day(2026, time.March, 1), day(2026, time.June, 30)

	tests := []struct {
		name  string
		today time.Time
		want  string
	}{
		{"day before start", day(2026, time.February, 28), helpers.ContractNotStarted},
		{"start day", mulai, helpers.ContractOngoing},
		{"middle", day(2026, time.May, 1), helpers.ContractOngoing},
		{"end day is still ongoing", selesai, helpers.ContractOngoing},
		{"day after end", day(2026, time.July, 1), helpers.ContractExpired},
		{"time of day is ignored", time.Date(2026, time.June, 30, 23, 59, 0, 0, time.UTC), helpers.ContractOngoing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, helpers.ContractStatus(mulai, selesai, tt.today))
		})
	}
}

func TestContractEndingSoon(t *testing.T) {
	mulai, selesai := day(2026, time.January, 1), day(2026, time.June, 30)

	assert.False(t, helpers.ContractEndingSoon(mulai, selesai, day(2026, time.May, 29)), "more than a month left")
	assert.True(t, helpers.ContractEndingSoon(mulai, selesai, day(2026, time.May, 30)), "exactly one month left")
	assert.True(t, helpers.ContractEndingSoon(mulai, selesai, selesai), "last day")
	assert.False(t, helpers.ContractEndingSoon(mulai, selesai, day(2026, time.July, 1)), "expired")
	assert.False(t, helpers.ContractEndingSoon(day(2026, time.July, 1), day(2026, time.July, 20), day(2026, time.June, 25)), "not started")
}

func TestContractDaysRemaining(t *testing.T) {
	selesai := day(2026, time.June, 30)

	assert.Equal(t, 10, helpers.ContractDaysRemaining(selesai, day(2026, time.June, 20)))
	assert.Equal(t, 0, helpers.ContractDaysRemaining(selesai, selesai))
	assert.Equal(t, 0, helpers.ContractDaysRemaining(selesai, day(2026, time.July, 5)), "never negative")
}
