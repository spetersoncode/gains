package client

import (
	"errors"
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

func TestUnmarshalError(t *testing.T) {
	t.Run("Error returns formatted message", func(t *testing.T) {
		err := &UnmarshalError{
			Content:    `{"invalid": json`,
			TargetType: "BookInfo",
			Err:        errors.New("unexpected end of JSON input"),
		}
		expected := "failed to unmarshal response into BookInfo: unexpected end of JSON input"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlying := errors.New("parse error")
		err := &UnmarshalError{
			Content:    "invalid",
			TargetType: "TestType",
			Err:        underlying,
		}
		assert.Equal(t, underlying, err.Unwrap())
		assert.True(t, errors.Is(err, underlying))
	})
}
