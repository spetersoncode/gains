package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"Book", "book"},
		{"BookInfo", "book_info"},
		{"SentimentAnalysis", "sentiment_analysis"},
		{"HTTPResponse", "h_t_t_p_response"},
		{"SimpleType", "simple_type"},
		{"A", "a"},
		{"Ab", "ab"},
		{"ABC", "a_b_c"},
		{"userID", "user_i_d"},
		{"lowercase", "lowercase"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := toSnakeCase(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
