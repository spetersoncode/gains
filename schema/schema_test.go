package schema

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestStringBuilder(t *testing.T) {
	tests := []struct {
		name    string
		builder Builder
		want    map[string]any
		wantErr error
	}{
		{
			name:    "basic string",
			builder: String(),
			want:    map[string]any{"type": "string"},
		},
		{
			name:    "string with description",
			builder: String().Desc("A name"),
			want:    map[string]any{"type": "string", "description": "A name"},
		},
		{
			name:    "string with enum",
			builder: String().Enum("a", "b", "c"),
			want:    map[string]any{"type": "string", "enum": []any{"a", "b", "c"}},
		},
		{
			name:    "string with length constraints",
			builder: String().MinLength(1).MaxLength(100),
			want:    map[string]any{"type": "string", "minLength": float64(1), "maxLength": float64(100)},
		},
		{
			name:    "string with pattern",
			builder: String().Pattern(`^[a-z]+$`),
			want:    map[string]any{"type": "string", "pattern": "^[a-z]+$"},
		},
		{
			name:    "string with default",
			builder: String().Default("hello"),
			want:    map[string]any{"type": "string", "default": "hello"},
		},
		{
			name:    "string with all constraints",
			builder: String().Desc("Test").MinLength(1).MaxLength(10).Pattern(`^\w+$`).Default("test"),
			want: map[string]any{
				"type":        "string",
				"description": "Test",
				"minLength":   float64(1),
				"maxLength":   float64(10),
				"pattern":     `^\w+$`,
				"default":     "test",
			},
		},
		{
			name:    "invalid minLength > maxLength",
			builder: String().MinLength(100).MaxLength(10),
			wantErr: ErrInvalidRange,
		},
		{
			name:    "invalid pattern",
			builder: String().Pattern(`[invalid`),
			wantErr: ErrInvalidPattern,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.builder.Build()
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, got, tt.want)
		})
	}
}

func TestIntBuilder(t *testing.T) {
	tests := []struct {
		name    string
		builder Builder
		want    map[string]any
		wantErr error
	}{
		{
			name:    "basic int",
			builder: Int(),
			want:    map[string]any{"type": "integer"},
		},
		{
			name:    "integer alias",
			builder: Integer(),
			want:    map[string]any{"type": "integer"},
		},
		{
			name:    "int with description",
			builder: Int().Desc("Count"),
			want:    map[string]any{"type": "integer", "description": "Count"},
		},
		{
			name:    "int with min/max",
			builder: Int().Min(1).Max(100),
			want:    map[string]any{"type": "integer", "minimum": float64(1), "maximum": float64(100)},
		},
		{
			name:    "int with exclusive bounds",
			builder: Int().ExclusiveMin(0).ExclusiveMax(100),
			want:    map[string]any{"type": "integer", "exclusiveMinimum": float64(0), "exclusiveMaximum": float64(100)},
		},
		{
			name:    "int with enum",
			builder: Int().Enum(1, 2, 3),
			want:    map[string]any{"type": "integer", "enum": []any{1, 2, 3}},
		},
		{
			name:    "int with default",
			builder: Int().Default(42),
			want:    map[string]any{"type": "integer", "default": 42},
		},
		{
			name:    "invalid min > max",
			builder: Int().Min(100).Max(10),
			wantErr: ErrInvalidRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.builder.Build()
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, got, tt.want)
		})
	}
}

func TestNumberBuilder(t *testing.T) {
	tests := []struct {
		name    string
		builder Builder
		want    map[string]any
		wantErr error
	}{
		{
			name:    "basic number",
			builder: Number(),
			want:    map[string]any{"type": "number"},
		},
		{
			name:    "number with min/max",
			builder: Number().Min(0.0).Max(1.0),
			want:    map[string]any{"type": "number", "minimum": 0.0, "maximum": 1.0},
		},
		{
			name:    "number with exclusive bounds",
			builder: Number().ExclusiveMin(0.0).ExclusiveMax(1.0),
			want:    map[string]any{"type": "number", "exclusiveMinimum": 0.0, "exclusiveMaximum": 1.0},
		},
		{
			name:    "number with enum",
			builder: Number().Enum(0.5, 1.0, 1.5),
			want:    map[string]any{"type": "number", "enum": []any{0.5, 1.0, 1.5}},
		},
		{
			name:    "invalid min > max",
			builder: Number().Min(1.0).Max(0.5),
			wantErr: ErrInvalidRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.builder.Build()
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, got, tt.want)
		})
	}
}

func TestBoolBuilder(t *testing.T) {
	tests := []struct {
		name    string
		builder Builder
		want    map[string]any
	}{
		{
			name:    "basic bool",
			builder: Bool(),
			want:    map[string]any{"type": "boolean"},
		},
		{
			name:    "boolean alias",
			builder: Boolean(),
			want:    map[string]any{"type": "boolean"},
		},
		{
			name:    "bool with description",
			builder: Bool().Desc("Is enabled"),
			want:    map[string]any{"type": "boolean", "description": "Is enabled"},
		},
		{
			name:    "bool with default true",
			builder: Bool().Default(true),
			want:    map[string]any{"type": "boolean", "default": true},
		},
		{
			name:    "bool with default false",
			builder: Bool().Default(false),
			want:    map[string]any{"type": "boolean", "default": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.builder.Build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, got, tt.want)
		})
	}
}

func TestArrayBuilder(t *testing.T) {
	tests := []struct {
		name    string
		builder Builder
		want    map[string]any
		wantErr error
	}{
		{
			name:    "array of strings",
			builder: Array(String()),
			want:    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		{
			name:    "array of integers",
			builder: Array(Int()),
			want:    map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
		},
		{
			name:    "array with description",
			builder: Array(String()).Desc("List of tags"),
			want:    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "List of tags"},
		},
		{
			name:    "array with min/max items",
			builder: Array(String()).MinItems(1).MaxItems(10),
			want:    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": float64(1), "maxItems": float64(10)},
		},
		{
			name:    "array with unique items",
			builder: Array(String()).UniqueItems(),
			want:    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "uniqueItems": true},
		},
		{
			name:    "array of objects",
			builder: Array(Object().Field("name", String())),
			want: map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":       "object",
					"properties": map[string]any{"name": map[string]any{"type": "string"}},
				},
			},
		},
		{
			name:    "invalid minItems > maxItems",
			builder: Array(String()).MinItems(10).MaxItems(1),
			wantErr: ErrInvalidRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.builder.Build()
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, got, tt.want)
		})
	}
}

func TestObjectBuilder(t *testing.T) {
	tests := []struct {
		name    string
		builder Builder
		want    map[string]any
		wantErr error
	}{
		{
			name:    "empty object",
			builder: Object(),
			want:    map[string]any{"type": "object"},
		},
		{
			name:    "object with description",
			builder: Object().Desc("A person"),
			want:    map[string]any{"type": "object", "description": "A person"},
		},
		{
			name:    "object with string field",
			builder: Object().Field("name", String()),
			want:    map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}}},
		},
		{
			name:    "object with required field",
			builder: Object().Field("name", String().Required()),
			want: map[string]any{
				"type":       "object",
				"properties": map[string]any{"name": map[string]any{"type": "string"}},
				"required":   []any{"name"},
			},
		},
		{
			name: "object with multiple fields",
			builder: Object().
				Field("name", String().Required()).
				Field("age", Int()).
				Field("active", Bool()),
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":   map[string]any{"type": "string"},
					"age":    map[string]any{"type": "integer"},
					"active": map[string]any{"type": "boolean"},
				},
				"required": []any{"name"},
			},
		},
		{
			name: "object with multiple required fields",
			builder: Object().
				Field("name", String().Required()).
				Field("email", String().Required()),
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":  map[string]any{"type": "string"},
					"email": map[string]any{"type": "string"},
				},
				"required": []any{"name", "email"},
			},
		},
		{
			name:    "object with strict mode",
			builder: Object().StrictMode().Field("name", String()),
			want: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"name": map[string]any{"type": "string"}},
				"additionalProperties": false,
			},
		},
		{
			name:    "object with additional properties true",
			builder: Object().AdditionalProperties(true).Field("name", String()),
			want: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"name": map[string]any{"type": "string"}},
				"additionalProperties": true,
			},
		},
		{
			name: "nested object",
			builder: Object().
				Field("user", Object().
					Field("name", String().Required()).
					Required()),
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"user": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{"type": "string"},
						},
						"required": []any{"name"},
					},
				},
				"required": []any{"user"},
			},
		},
		{
			name: "object with array field",
			builder: Object().
				Field("tags", Array(String()).MaxItems(10)),
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tags": map[string]any{
						"type":     "array",
						"items":    map[string]any{"type": "string"},
						"maxItems": float64(10),
					},
				},
			},
		},
		{
			name: "validation propagates from nested field",
			builder: Object().
				Field("count", Int().Min(100).Max(10)),
			wantErr: ErrInvalidRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.builder.Build()
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, got, tt.want)
		})
	}
}

func TestMustBuild(t *testing.T) {
	// Valid schema should not panic
	t.Run("valid schema", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic: %v", r)
			}
		}()
		_ = String().MustBuild()
	})

	// Invalid schema should panic
	t.Run("invalid schema panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic, got none")
			}
		}()
		_ = String().MinLength(100).MaxLength(10).MustBuild()
	})
}

func TestRequiredFieldDuplicates(t *testing.T) {
	// Adding the same field twice as required should not duplicate in required array
	obj := Object().
		Field("name", String().Required()).
		Field("name", String().Required())

	got, err := obj.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	required, ok := result["required"].([]any)
	if !ok {
		t.Fatal("required should be an array")
	}

	// Count occurrences of "name"
	count := 0
	for _, r := range required {
		if r == "name" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 occurrence of 'name' in required, got %d", count)
	}
}

func TestComplexSchema(t *testing.T) {
	// Test a complex real-world schema
	schema := Object().
		Desc("Weather forecast request").
		Field("location", String().Desc("City name").MinLength(1).MaxLength(100).Required()).
		Field("unit", String().Enum("celsius", "fahrenheit").Default("celsius")).
		Field("days", Int().Desc("Number of days").Min(1).Max(14).Default(7)).
		Field("include", Object().
			Field("humidity", Bool().Default(true)).
			Field("wind", Bool().Default(false)).
			Field("uv_index", Bool())).
		Field("coordinates", Object().
			Field("lat", Number().Min(-90).Max(90).Required()).
			Field("lon", Number().Min(-180).Max(180).Required())).
		Field("tags", Array(String()).MaxItems(5).UniqueItems()).
		StrictMode()

	got, err := schema.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify key aspects
	if result["type"] != "object" {
		t.Errorf("expected type 'object', got %v", result["type"])
	}
	if result["description"] != "Weather forecast request" {
		t.Errorf("expected description, got %v", result["description"])
	}
	if result["additionalProperties"] != false {
		t.Errorf("expected additionalProperties false, got %v", result["additionalProperties"])
	}

	props := result["properties"].(map[string]any)
	if len(props) != 6 {
		t.Errorf("expected 6 properties, got %d", len(props))
	}

	required := result["required"].([]any)
	if len(required) != 1 || required[0] != "location" {
		t.Errorf("expected required ['location'], got %v", required)
	}
}

// assertJSONEqual compares a json.RawMessage to an expected map structure.
func assertJSONEqual(t *testing.T, got json.RawMessage, want map[string]any) {
	t.Helper()

	var gotMap map[string]any
	if err := json.Unmarshal(got, &gotMap); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	wantJSON, _ := json.Marshal(want)
	gotJSON, _ := json.Marshal(gotMap)

	if string(gotJSON) != string(wantJSON) {
		t.Errorf("JSON mismatch:\ngot:  %s\nwant: %s", gotJSON, wantJSON)
	}
}
