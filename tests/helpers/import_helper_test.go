package helpers_test

import (
	"testing"

	"gatra-budget-backend/helpers"

	"github.com/stretchr/testify/assert"
)

func TestParseImportAmount(t *testing.T) {
	for _, ok := range []string{"1500000", " 1500000.50 ", "0"} {
		_, valid := helpers.ParseImportAmount(ok)
		assert.True(t, valid, ok)
	}
	for _, bad := range []string{"", "abc", "1.500.000", "NaN", "Inf"} {
		_, valid := helpers.ParseImportAmount(bad)
		assert.False(t, valid, bad)
	}
}

func TestParseImportCount(t *testing.T) {
	v, ok := helpers.ParseImportCount(" 12 ")
	assert.True(t, ok)
	assert.Equal(t, 12, v)

	for _, bad := range []string{"", "-1", "1.5", "x"} {
		_, ok := helpers.ParseImportCount(bad)
		assert.False(t, ok, bad)
	}
}

func TestNormalizeKey(t *testing.T) {
	assert.Equal(t, "pt maju jaya", helpers.NormalizeKey("  PT Maju Jaya "))
}

func TestTranslateValidationMessage(t *testing.T) {
	tests := []struct {
		in, field, message string
	}{
		{"no_sp2k is required", "no_sp2k", "wajib diisi"},
		{"nilai must be greater than 0", "nilai", "harus lebih dari 0"},
		{"tgl_bayar is required when status is Clearing", "tgl_bayar", "wajib diisi bila status Clearing"},
		{"invalid tanggal_mulai: invalid date format, expected YYYY-MM-DD", "tanggal_mulai", "format tanggal tidak valid, gunakan YYYY-MM-DD"},
		{"tanggal_selesai harus setelah tanggal_mulai", "tanggal_selesai", "harus setelah tanggal_mulai"},
		{"manajemen_fee tidak boleh melebihi nilai", "manajemen_fee", "tidak boleh melebihi nilai"},
	}
	for _, tt := range tests {
		field, message := helpers.TranslateValidationMessage(tt.in)
		assert.Equal(t, tt.field, field, tt.in)
		assert.Equal(t, tt.message, message, tt.in)
	}
}
