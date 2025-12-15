package gains

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testModel is a simple Model implementation for testing.
type testModel string

func (m testModel) String() string { return string(m) }

func TestApplyOptions(t *testing.T) {
	t.Run("returns empty options when no options provided", func(t *testing.T) {
		opts := ApplyOptions()
		assert.NotNil(t, opts)
		assert.Nil(t, opts.Model)
		assert.Zero(t, opts.MaxTokens)
		assert.Nil(t, opts.Temperature)
		assert.Nil(t, opts.Tools)
		assert.Empty(t, opts.ToolChoice)
		assert.Empty(t, opts.ResponseFormat)
		assert.Nil(t, opts.ResponseSchema)
	})

	t.Run("applies multiple options", func(t *testing.T) {
		tools := []Tool{{Name: "test"}}
		opts := ApplyOptions(
			WithModel(testModel("gpt-4")),
			WithMaxTokens(1000),
			WithTemperature(0.7),
			WithTools(tools),
			WithToolChoice(ToolChoiceRequired),
		)

		assert.Equal(t, "gpt-4", opts.Model.String())
		assert.Equal(t, 1000, opts.MaxTokens)
		require.NotNil(t, opts.Temperature)
		assert.Equal(t, 0.7, *opts.Temperature)
		assert.Equal(t, tools, opts.Tools)
		assert.Equal(t, ToolChoiceRequired, opts.ToolChoice)
	})
}

func TestWithModel(t *testing.T) {
	tests := []struct {
		name     string
		model    testModel
		expected string
	}{
		{"sets gpt-4", "gpt-4", "gpt-4"},
		{"sets claude-3-opus", "claude-3-opus", "claude-3-opus"},
		{"sets gemini-pro", "gemini-pro", "gemini-pro"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyOptions(WithModel(tt.model))
			assert.Equal(t, tt.expected, opts.Model.String())
		})
	}
}

func TestWithMaxTokens(t *testing.T) {
	tests := []struct {
		name     string
		tokens   int
		expected int
	}{
		{"sets positive value", 1000, 1000},
		{"sets zero", 0, 0},
		{"sets large value", 100000, 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyOptions(WithMaxTokens(tt.tokens))
			assert.Equal(t, tt.expected, opts.MaxTokens)
		})
	}
}

func TestWithTemperature(t *testing.T) {
	tests := []struct {
		name     string
		temp     float64
		expected float64
	}{
		{"sets zero", 0.0, 0.0},
		{"sets mid value", 0.7, 0.7},
		{"sets max value", 2.0, 2.0},
		{"sets fractional", 0.123, 0.123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyOptions(WithTemperature(tt.temp))
			require.NotNil(t, opts.Temperature)
			assert.Equal(t, tt.expected, *opts.Temperature)
		})
	}
}

func TestWithTools(t *testing.T) {
	t.Run("sets tools slice", func(t *testing.T) {
		tools := []Tool{
			{Name: "get_weather", Description: "Get weather"},
			{Name: "search", Description: "Search the web"},
		}
		opts := ApplyOptions(WithTools(tools))
		assert.Equal(t, tools, opts.Tools)
		assert.Len(t, opts.Tools, 2)
	})

	t.Run("sets empty slice", func(t *testing.T) {
		opts := ApplyOptions(WithTools([]Tool{}))
		assert.Empty(t, opts.Tools)
	})

	t.Run("sets nil slice", func(t *testing.T) {
		opts := ApplyOptions(WithTools(nil))
		assert.Nil(t, opts.Tools)
	})
}

func TestWithToolChoice(t *testing.T) {
	tests := []struct {
		name     string
		choice   ToolChoice
		expected ToolChoice
	}{
		{"sets auto", ToolChoiceAuto, ToolChoiceAuto},
		{"sets none", ToolChoiceNone, ToolChoiceNone},
		{"sets required", ToolChoiceRequired, ToolChoiceRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyOptions(WithToolChoice(tt.choice))
			assert.Equal(t, tt.expected, opts.ToolChoice)
		})
	}
}

func TestWithJSONMode(t *testing.T) {
	t.Run("sets response format to JSON", func(t *testing.T) {
		opts := ApplyOptions(WithJSONMode())
		assert.Equal(t, ResponseFormatJSON, opts.ResponseFormat)
	})
}

func TestWithResponseSchema(t *testing.T) {
	t.Run("sets schema and enables JSON mode", func(t *testing.T) {
		schema := ResponseSchema{
			Name:        "TestSchema",
			Description: "A test schema",
			Schema:      json.RawMessage(`{"type":"object"}`),
			Strict:      true,
		}
		opts := ApplyOptions(WithResponseSchema(schema))

		assert.Equal(t, ResponseFormatJSON, opts.ResponseFormat)
		require.NotNil(t, opts.ResponseSchema)
		assert.Equal(t, "TestSchema", opts.ResponseSchema.Name)
		assert.Equal(t, "A test schema", opts.ResponseSchema.Description)
		assert.True(t, opts.ResponseSchema.Strict)
	})

	t.Run("later option overrides earlier", func(t *testing.T) {
		opts := ApplyOptions(
			WithModel(testModel("first")),
			WithModel(testModel("second")),
		)
		assert.Equal(t, "second", opts.Model.String())
	})
}

func TestResponseFormatConstants(t *testing.T) {
	assert.Equal(t, ResponseFormat("text"), ResponseFormatText)
	assert.Equal(t, ResponseFormat("json"), ResponseFormatJSON)
}
