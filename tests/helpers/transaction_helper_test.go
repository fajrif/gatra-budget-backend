package helpers_test

import (
	"testing"
	"time"

	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/stretchr/testify/assert"
)

func date(year int, month time.Month, day int) models.Date {
	return models.Date{Time: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

func TestComputeBulanRealisasi(t *testing.T) {
	result := helpers.ComputeBulanRealisasi(date(2024, time.March, 15))
	assert.Equal(t, "2024-03", result)
}

func TestComputeManagementFee(t *testing.T) {
	t.Run("nil rate yields nil fields", func(t *testing.T) {
		persentase, nilai := helpers.ComputeManagementFee(1_000_000, nil)
		assert.Nil(t, persentase)
		assert.Nil(t, nilai)
	})

	t.Run("present rate sets both fields together", func(t *testing.T) {
		tarif := 10.0
		persentase, nilai := helpers.ComputeManagementFee(500_000_000, &tarif)
		assert.NotNil(t, persentase)
		assert.NotNil(t, nilai)
		assert.Equal(t, 10.0, *persentase)
		assert.Equal(t, 50_000_000.0, *nilai)
	})

	t.Run("rounds to 2 decimal places", func(t *testing.T) {
		tarif := 7.5
		_, nilai := helpers.ComputeManagementFee(333_333.33, &tarif)
		assert.Equal(t, 25000.0, *nilai)
	})
}

func TestTransactionSLAStatus(t *testing.T) {
	tests := []struct {
		name             string
		tanggalTransaksi models.Date
		tanggalPOTagihan *models.Date
		expected         string
	}{
		{
			name:             "no PO date is no_data",
			tanggalTransaksi: date(2024, time.March, 10),
			tanggalPOTagihan: nil,
			expected:         "no_data",
		},
		{
			name:             "same month as PO date is on_time",
			tanggalTransaksi: date(2024, time.March, 28),
			tanggalPOTagihan: ptr(date(2024, time.March, 1)),
			expected:         "on_time",
		},
		{
			name:             "transaction before PO month is on_time",
			tanggalTransaksi: date(2024, time.February, 28),
			tanggalPOTagihan: ptr(date(2024, time.March, 1)),
			expected:         "on_time",
		},
		{
			name:             "exactly one month after PO date is h_plus_1",
			tanggalTransaksi: date(2024, time.April, 5),
			tanggalPOTagihan: ptr(date(2024, time.March, 1)),
			expected:         "h_plus_1",
		},
		{
			name:             "more than one month after PO date is late",
			tanggalTransaksi: date(2024, time.May, 1),
			tanggalPOTagihan: ptr(date(2024, time.March, 1)),
			expected:         "late",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.TransactionSLAStatus(tt.tanggalTransaksi, tt.tanggalPOTagihan)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func ptr(d models.Date) *models.Date {
	return &d
}
