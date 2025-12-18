package client

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	ai "github.com/spetersoncode/gains"
)

// ChatTyped sends a chat request and unmarshals the response into type T.
// The response schema is automatically generated from the struct type.
//
// This is a convenience function that combines WithResponseSchema and
// json.Unmarshal into a single call:
//
//	// Instead of:
//	resp, err := c.Chat(ctx, msgs, ai.WithResponseSchema(ai.ResponseSchema{
//	    Name: "book_info", Schema: ai.MustSchemaFor[BookInfo](),
//	}))
//	var book BookInfo
//	json.Unmarshal([]byte(resp.Content), &book)
//
//	// You can use:
//	book, err := client.ChatTyped[BookInfo](ctx, c, msgs)
//
// The schema name is derived from the type name using snake_case conversion.
// All provided options are passed through to the underlying Chat call.
func ChatTyped[T any](ctx context.Context, c *Client, msgs []ai.Message, opts ...ai.Option) (T, error) {
	var zero T

	// Get type information
	t := reflect.TypeOf(zero)
	if t == nil {
		return zero, fmt.Errorf("ChatTyped: cannot use nil type")
	}

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Generate schema name from type name
	schemaName := toSnakeCase(t.Name())
	if schemaName == "" {
		schemaName = "response"
	}

	// Generate schema from type
	schema, err := ai.SchemaFor[T]()
	if err != nil {
		return zero, fmt.Errorf("ChatTyped: failed to generate schema: %w", err)
	}

	// Build response schema option
	responseSchema := ai.ResponseSchema{
		Name:   schemaName,
		Schema: schema,
	}

	// Prepend the response schema option so user opts can override if needed
	allOpts := make([]ai.Option, 0, len(opts)+1)
	allOpts = append(allOpts, ai.WithResponseSchema(responseSchema))
	allOpts = append(allOpts, opts...)

	// Make the chat request
	resp, err := c.Chat(ctx, msgs, allOpts...)
	if err != nil {
		return zero, err
	}

	// Unmarshal the response
	var result T
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return zero, &UnmarshalError{
			Content:    resp.Content,
			TargetType: t.String(),
			Err:        err,
		}
	}

	return result, nil
}

// UnmarshalError is returned when the LLM response cannot be unmarshaled
// into the target type.
type UnmarshalError struct {
	Content    string
	TargetType string
	Err        error
}

func (e *UnmarshalError) Error() string {
	return fmt.Sprintf("failed to unmarshal response into %s: %v", e.TargetType, e.Err)
}

func (e *UnmarshalError) Unwrap() error {
	return e.Err
}

// toSnakeCase converts a CamelCase string to snake_case.
func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
