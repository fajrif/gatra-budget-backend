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
