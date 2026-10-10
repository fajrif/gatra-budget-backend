package services_test

import (
	"testing"

	"gatra-budget-backend/services"

	"github.com/stretchr/testify/assert"
)

func sum(amounts []float64) float64 {
	var total float64
	for _, a := range amounts {
		total += a
	}
	return total
}

func TestSplitAmount(t *testing.T) {
	t.Run("divides evenly with no remainder", func(t *testing.T) {
		amounts := services.SplitAmount(1_200_000, 12)
		assert.Len(t, amounts, 12)
		for _, a := range amounts {
			assert.Equal(t, 100_000.0, a)
		}
		assert.Equal(t, 1_200_000.0, sum(amounts))
	})

	t.Run("pushes rounding remainder onto the last part", func(t *testing.T) {
		amounts := services.SplitAmount(1_000_000, 6)
		assert.Len(t, amounts, 6)
		for _, a := range amounts[:5] {
			assert.Equal(t, 166_666.67, a)
		}
		assert.Equal(t, 166_666.65, amounts[5])
		assert.InDelta(t, 1_000_000.0, sum(amounts), 0.01)
	})

	t.Run("single month returns the whole total", func(t *testing.T) {
		amounts := services.SplitAmount(500_000, 1)
		assert.Equal(t, []float64{500_000}, amounts)
	})

	t.Run("n <= 0 returns nil", func(t *testing.T) {
		assert.Nil(t, services.SplitAmount(500_000, 0))
		assert.Nil(t, services.SplitAmount(500_000, -1))
	})

	t.Run("children always sum exactly to the total, across varied inputs", func(t *testing.T) {
		cases := []struct {
			total float64
			n     int
		}{
			{100_000_000, 12}, {1, 3}, {999_999.99, 7}, {333_333.33, 4}, {0, 5},
		}
		for _, c := range cases {
			amounts := services.SplitAmount(c.total, c.n)
			assert.InDelta(t, c.total, sum(amounts), 0.01, "total=%v n=%v", c.total, c.n)
		}
	})
}
