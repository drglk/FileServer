package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"valid number", "10", 10},
		{"zero", "0", 0},
		{"negative number", "-5", 0},
		{"not a number", "abc", 0},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ParseLimit(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
