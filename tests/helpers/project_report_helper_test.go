package helpers_test

import (
	"testing"

	"gatra-budget-backend/helpers"

	"github.com/stretchr/testify/assert"
)

func TestCalculateAbsorption(t *testing.T) {
	t.Run("zero contract value guards against division by zero", func(t *testing.T) {
		pct, alert := helpers.CalculateAbsorption(0, 500_000)
		assert.Equal(t, 0.0, pct)
		assert.False(t, alert)
	})

	t.Run("below threshold", func(t *testing.T) {
		pct, alert := helpers.CalculateAbsorption(1_000_000_000, 700_000_000)
		assert.Equal(t, 70.0, pct)
		assert.False(t, alert)
	})

	t.Run("above threshold triggers alert", func(t *testing.T) {
		pct, alert := helpers.CalculateAbsorption(1_000_000_000, 850_000_000)
		assert.Equal(t, 85.0, pct)
		assert.True(t, alert)
	})

	t.Run("exactly at threshold does not trigger alert", func(t *testing.T) {
		_, alert := helpers.CalculateAbsorption(1_000_000_000, 800_000_000)
		assert.False(t, alert)
	})
}

func TestCalculateRemainingBudget(t *testing.T) {
	result := helpers.CalculateRemainingBudget(1_000_000_000, 650_000_000)
	assert.Equal(t, 350_000_000.0, result)
}

func TestCalculateProfitMargin(t *testing.T) {
	t.Run("nil rate yields zero margin", func(t *testing.T) {
		result := helpers.CalculateProfitMargin(nil, 500_000_000)
		assert.Equal(t, 0.0, result)
	})

	t.Run("present rate computes fee-style margin", func(t *testing.T) {
		tarif := 10.0
		result := helpers.CalculateProfitMargin(&tarif, 500_000_000)
		assert.Equal(t, 50_000_000.0, result)
	})
}
