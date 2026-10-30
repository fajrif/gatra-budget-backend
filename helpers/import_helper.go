package helpers

import (
	"math"
	"strconv"
	"strings"
)

// MaxImportRows caps how many rows one Excel import may carry.
const MaxImportRows = 1000

// ImportRowError is one problem found in an import row. Row is the row number
// in the Excel sheet so the user can find it.
type ImportRowError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// NormalizeKey is how names and codes are matched during import: trimmed and
// case-insensitive.
func NormalizeKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ParseImportAmount parses an amount cell sent as text. Only plain numbers are
// accepted (no thousand separators).
func ParseImportAmount(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// ParseImportCount parses a non-negative whole number cell.
func ParseImportCount(s string) (int, bool) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v < 0 {
		return 0, false
	}
	return v, true
}

// validationMessages maps the English messages of the Validate*Request
// functions to what the import shows the user.
var validationMessages = map[string]string{
	"jumlah_realisasi is required and must be greater than 100,000": "harus lebih dari 100.000",
	"nilai must be greater than 0":                                  "harus lebih dari 0",
	"status must be 'Submit' or 'Clearing'":                         "harus 'Submit' atau 'Clearing'",
	"tgl_bayar is required when status is Clearing":                 "wajib diisi bila status Clearing",
	"tgl_bayar must be empty when status is Submit":                 "harus kosong bila status Submit",
	"manajemen_fee must not be negative":                            "tidak boleh negatif",
}

// TranslateValidationMessage splits a validator message ("<field> <rule>") into
// the field and an Indonesian description of the rule. Messages without a known
// translation are passed through unchanged.
func TranslateValidationMessage(msg string) (field, message string) {
	// apply*Request errors read "invalid <field>: <reason>" (a bad date).
	if bad, ok := strings.CutPrefix(msg, "invalid "); ok {
		field, _, _ := strings.Cut(bad, ":")
		return field, "format tanggal tidak valid, gunakan YYYY-MM-DD"
	}

	field, rest, found := strings.Cut(msg, " ")
	if !found {
		return "", msg
	}
	if translated, ok := validationMessages[msg]; ok {
		return field, translated
	}
	if rest == "is required" {
		return field, "wajib diisi"
	}
	return field, rest
}
