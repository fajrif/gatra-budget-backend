package helpers_test

import (
	"boilerplates-backend/helpers"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateDurationInMonths(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected int
	}{
		{
			name:     "Same month and year returns 0",
			start:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "Exactly one month difference",
			start:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "One month but end day before start day",
			start:    time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "Multiple months in same year",
			start:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			expected: 5,
		},
		{
			name:     "Across year boundary",
			start:    time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			expected: 3,
		},
		{
			name:     "Multiple years",
			start:    time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: 24,
		},
		{
			name:     "End date before start date returns 0",
			start:    time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "Same date returns 0",
			start:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "Edge case - month end to month start",
			start:    time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "Leap year February",
			start:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.CalculateDurationInMonths(tt.start, tt.end)
			assert.Equal(t, tt.expected, result, "Expected %d months but got %d", tt.expected, result)
		})
	}
}

func TestCalculateDurationInMonths_EdgeCases(t *testing.T) {
	t.Run("Very long duration", func(t *testing.T) {
		start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		result := helpers.CalculateDurationInMonths(start, end)
		assert.Equal(t, 300, result)
	})

	t.Run("Different timezones same actual time", func(t *testing.T) {
		loc1, _ := time.LoadLocation("America/New_York")
		loc2, _ := time.LoadLocation("Asia/Tokyo")
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, loc1)
		end := time.Date(2024, 3, 1, 0, 0, 0, 0, loc2)
		result := helpers.CalculateDurationInMonths(start, end)
		assert.GreaterOrEqual(t, result, 0)
	})
}
