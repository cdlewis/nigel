package main

import (
	"testing"
	"time"
)

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name     string
		level    int
		expected time.Duration
	}{
		{
			name:     "level 0 returns base backoff",
			level:    0,
			expected: 5 * time.Minute,
		},
		{
			name:     "level 1 doubles",
			level:    1,
			expected: 10 * time.Minute,
		},
		{
			name:     "level 2 quadruples",
			level:    2,
			expected: 20 * time.Minute,
		},
		{
			name:     "level 3 is 40 minutes",
			level:    3,
			expected: 40 * time.Minute,
		},
		{
			name:     "level 4 caps at max",
			level:    4,
			expected: 1 * time.Hour,
		},
		{
			name:     "level 5 stays at max",
			level:    5,
			expected: 1 * time.Hour,
		},
		{
			name:     "level 10 stays at max",
			level:    10,
			expected: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateBackoff(tt.level)
			if result != tt.expected {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.level, result, tt.expected)
			}
		})
	}
}

func TestRateLimitError(t *testing.T) {
	err := &rateLimitError{msg: "test rate limit"}

	if err.Error() != "test rate limit" {
		t.Errorf("rateLimitError.Error() = %q, want %q", err.Error(), "test rate limit")
	}

	// Verify it implements error interface
	var _ error = err
}
