package gains

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaFrom_SimpleTypes(t *testing.T) {
	type Args struct {
		Name    string  `json:"name"`
		Age     int     `json:"age"`
		Score   float64 `json:"score"`
		Active  bool    `json:"active"`
		Count   int64   `json:"count"`
		Rating  float32 `json:"rating"`
		SmallID uint8   `json:"small_id"`
	}

	schema := SchemaFrom[Args]().Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	assert.Equal(t, "object", result["type"])
	props := result["properties"].(map[string]any)

	assert.Equal(t, "string", props["name"].(map[string]any)["type"])
	assert.Equal(t, "integer", props["age"].(map[string]any)["type"])
	assert.Equal(t, "number", props["score"].(map[string]any)["type"])
	assert.Equal(t, "boolean", props["active"].(map[string]any)["type"])
	assert.Equal(t, "integer", props["count"].(map[string]any)["type"])
	assert.Equal(t, "number", props["rating"].(map[string]any)["type"])
	assert.Equal(t, "integer", props["small_id"].(map[string]any)["type"])
}

func TestSchemaFrom_Required(t *testing.T) {
	type Args struct {
		Location string `json:"location"`
		Unit     string `json:"unit"`
	}

	schema := SchemaFrom[Args]().
		Required("location").
		Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	required := result["required"].([]any)
	assert.Len(t, required, 1)
	assert.Equal(t, "location", required[0])
}

func TestSchemaFrom_MultipleRequired(t *testing.T) {
	type Args struct {
		A string `json:"a"`
		B string `json:"b"`
		C string `json:"c"`
	}

	schema := SchemaFrom[Args]().
		Required("a", "c").
		Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	required := result["required"].([]any)
	assert.Len(t, required, 2)
	assert.Contains(t, required, "a")
	assert.Contains(t, required, "c")
}

func TestSchemaFrom_Desc(t *testing.T) {
	type Args struct {
		Query string `json:"query"`
	}

	schema := SchemaFrom[Args]().
		Desc("query", "The search query").
		Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	props := result["properties"].(map[string]any)
	assert.Equal(t, "The search query", props["query"].(map[string]any)["description"])
}

func TestSchemaFrom_Enum(t *testing.T) {
	type Args struct {
		Unit string `json:"unit"`
	}

	schema := SchemaFrom[Args]().
		Enum("unit", "celsius", "fahrenheit").
		Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	props := result["properties"].(map[string]any)
	enum := props["unit"].(map[string]any)["enum"].([]any)
	assert.Len(t, enum, 2)
	assert.Contains(t, enum, "celsius")
	assert.Contains(t, enum, "fahrenheit")
}

func TestSchemaFrom_Array(t *testing.T) {
	type Args struct {
		Tags []string `json:"tags"`
	}

	schema := SchemaFrom[Args]().Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	props := result["properties"].(map[string]any)
	tags := props["tags"].(map[string]any)
	assert.Equal(t, "array", tags["type"])

	items := tags["items"].(map[string]any)
	assert.Equal(t, "string", items["type"])
}

func TestSchemaFrom_ArrayOfInts(t *testing.T) {
	type Args struct {
		Numbers []int `json:"numbers"`
	}

	schema := SchemaFrom[Args]().Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	props := result["properties"].(map[string]any)
	numbers := props["numbers"].(map[string]any)
	assert.Equal(t, "array", numbers["type"])

	items := numbers["items"].(map[string]any)
	assert.Equal(t, "integer", items["type"])
}

func TestSchemaFrom_NestedStruct(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}

	type Args struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	schema := SchemaFrom[Args]().Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	props := result["properties"].(map[string]any)

	// Name is a string
	assert.Equal(t, "string", props["name"].(map[string]any)["type"])

	// Address is a nested object
	addr := props["address"].(map[string]any)
	assert.Equal(t, "object", addr["type"])

	addrProps := addr["properties"].(map[string]any)
	assert.Equal(t, "string", addrProps["street"].(map[string]any)["type"])
	assert.Equal(t, "string", addrProps["city"].(map[string]any)["type"])
}

func TestSchemaFrom_EmptyStruct(t *testing.T) {
	type Args struct{}

	schema := SchemaFrom[Args]().Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	assert.Equal(t, "object", result["type"])
	props := result["properties"].(map[string]any)
	assert.Empty(t, props)
}

func TestSchemaFrom_PointerFields(t *testing.T) {
	type Args struct {
		Name *string `json:"name"`
		Age  *int    `json:"age"`
	}

	schema := SchemaFrom[Args]().Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	props := result["properties"].(map[string]any)
	assert.Equal(t, "string", props["name"].(map[string]any)["type"])
	assert.Equal(t, "integer", props["age"].(map[string]any)["type"])
}

func TestSchemaFrom_JsonTagOmit(t *testing.T) {
	type Args struct {
		Public  string `json:"public"`
		Private string `json:"-"`
	}

	schema := SchemaFrom[Args]().Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	props := result["properties"].(map[string]any)
	assert.Contains(t, props, "public")
	assert.NotContains(t, props, "Private")
	assert.NotContains(t, props, "-")
}

func TestSchemaFrom_UnexportedFields(t *testing.T) {
	type Args struct {
		Public  string `json:"public"`
		private string //nolint:unused
	}

	schema := SchemaFrom[Args]().Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	props := result["properties"].(map[string]any)
	assert.Len(t, props, 1)
	assert.Contains(t, props, "public")
}

func TestSchemaFrom_ChainedMethods(t *testing.T) {
	type WeatherArgs struct {
		Location string `json:"location"`
		Unit     string `json:"unit"`
	}

	schema := SchemaFrom[WeatherArgs]().
		Desc("location", "The city name").
		Required("location").
		Desc("unit", "Temperature unit").
		Enum("unit", "celsius", "fahrenheit").
		Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	props := result["properties"].(map[string]any)

	// Check location
	location := props["location"].(map[string]any)
	assert.Equal(t, "string", location["type"])
	assert.Equal(t, "The city name", location["description"])

	// Check unit
	unit := props["unit"].(map[string]any)
	assert.Equal(t, "string", unit["type"])
	assert.Equal(t, "Temperature unit", unit["description"])
	enum := unit["enum"].([]any)
	assert.Contains(t, enum, "celsius")
	assert.Contains(t, enum, "fahrenheit")

	// Check required
	required := result["required"].([]any)
	assert.Contains(t, required, "location")
}

func TestSchemaFrom_InvalidFieldIgnored(t *testing.T) {
	type Args struct {
		Name string `json:"name"`
	}

	// These should not panic, just be no-ops
	schema := SchemaFrom[Args]().
		Desc("nonexistent", "description").
		Required("nonexistent").
		Enum("nonexistent", "a", "b").
		Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	// Should still have the valid field
	props := result["properties"].(map[string]any)
	assert.Contains(t, props, "name")

	// Should not have required since the field doesn't exist
	_, hasRequired := result["required"]
	assert.False(t, hasRequired)
}

func TestSchemaFrom_ArrayOfStructs(t *testing.T) {
	type Item struct {
		Name  string `json:"name"`
		Price int    `json:"price"`
	}

	type Args struct {
		Items []Item `json:"items"`
	}

	schema := SchemaFrom[Args]().Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	props := result["properties"].(map[string]any)
	items := props["items"].(map[string]any)
	assert.Equal(t, "array", items["type"])

	itemSchema := items["items"].(map[string]any)
	assert.Equal(t, "object", itemSchema["type"])

	itemProps := itemSchema["properties"].(map[string]any)
	assert.Equal(t, "string", itemProps["name"].(map[string]any)["type"])
	assert.Equal(t, "integer", itemProps["price"].(map[string]any)["type"])
}

func TestSchemaFrom_DuplicateRequired(t *testing.T) {
	type Args struct {
		Name string `json:"name"`
	}

	// Calling Required multiple times with same field shouldn't duplicate
	schema := SchemaFrom[Args]().
		Required("name").
		Required("name").
		Build()

	var result map[string]any
	err := json.Unmarshal(schema, &result)
	require.NoError(t, err)

	required := result["required"].([]any)
	assert.Len(t, required, 1)
}
