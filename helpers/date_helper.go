// src/helpers/date_helper.go (buat file baru untuk ini)
package helpers

import (
	"fmt"
	"time"
)

const DateFormat = "2006-01-02"

func ParseDateString(dateStr *string) (*time.Time, error) {
	if dateStr == nil || *dateStr == "" {
		return nil, nil
	}

	parsed, err := time.Parse(DateFormat, *dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD")
	}
	return &parsed, nil
}

func MustParseDateString(dateStr *string) *time.Time {
	parsed, _ := ParseDateString(dateStr)
	return parsed
}

// CalculateDurationInMonths menghitung durasi antara start dan end date dalam bulan.
func CalculateDurationInMonths(start, end time.Time) int {
	// Pastikan end date setelah start date (validasi sudah dilakukan sebelumnya, tapi baiknya cek lagi)
	if end.Before(start) {
		return 0
	}

	// Hitung perbedaan tahun dan bulan
	years := end.Year() - start.Year()
	months := int(end.Month()) - int(start.Month())

	// Hitung total bulan
	totalMonths := years*12 + months

	// Aturan bisnis: Jika dalam kurun bulan yang sama, diisi 0.
	// Jika totalMonths adalah 0, ini berarti tahun dan bulan sama.
	if totalMonths == 0 {
		return 0
	}

	// Sesuaikan jika tanggal selesai sebelum tanggal mulai di bulan yang sama (kasus edge)
	if end.Day() < start.Day() && totalMonths > 0 {
		totalMonths--
	}

	return totalMonths
}
