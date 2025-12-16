// Package schema provides a fluent API for building JSON Schema objects
// for use with AI tool parameters and structured output.
//
// Unlike reflection-based approaches, this package uses pure programmatic
// construction with compile-time type safety and build-time validation.
//
// # Basic Usage
//
// Create schemas using the type constructors and chain constraint methods:
//
//	params := schema.Object().
//		Field("location", schema.String().Desc("City name").Required()).
//		Field("unit", schema.String().Enum("celsius", "fahrenheit")).
//		Field("days", schema.Int().Min(1).Max(14).Default(7)).
//		MustBuild()
//
// # With Tool Definitions
//
//	tool := gains.Tool{
//		Name:        "get_forecast",
//		Description: "Get weather forecast",
//		Parameters: schema.Object().
//			Field("location", schema.String().Required()).
//			Field("days", schema.Int().Min(1).Max(14)).
//			MustBuild(),
//	}
//
// # Response Schemas
//
//	resp := gains.ResponseSchema{
//		Name: "book_info",
//		Schema: schema.Object().
//			Field("title", schema.String().Required()).
//			Field("year", schema.Int().Min(1000).Max(2100)).
//			MustBuild(),
//	}
//
// # Nested Objects
//
//	params := schema.Object().
//		Field("user", schema.Object().
//			Field("name", schema.String().Required()).
//			Field("age", schema.Int().Min(0)).
//			Required()).
//		Field("tags", schema.Array(schema.String()).MaxItems(10)).
//		MustBuild()
//
// # String Constraints
//
//	schema.String().
//		MinLength(1).
//		MaxLength(100).
//		Pattern(`^[a-z]+$`).
//		Build()
//
// # Numeric Constraints
//
//	schema.Int().Min(1).Max(100).Build()
//	schema.Number().ExclusiveMin(0).ExclusiveMax(1.0).Build()
//
// # Array Constraints
//
//	schema.Array(schema.String()).
//		MinItems(1).
//		MaxItems(10).
//		UniqueItems().
//		Build()
//
// # Validation
//
// Use Build() instead of MustBuild() to handle errors:
//
//	params, err := schema.Object().
//		Field("count", schema.Int().Min(10).Max(5)). // Error: min > max
//		Build()
//	if err != nil {
//		log.Fatal(err) // schema: minimum exceeds maximum
//	}
//
// # OpenAI Strict Mode
//
// For OpenAI compatibility, use StrictMode() which sets additionalProperties to false:
//
//	params := schema.Object().
//		StrictMode().
//		Field("name", schema.String().Required()).
//		MustBuild()
package schema
