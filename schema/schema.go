package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

// Builder is the interface implemented by all schema builders.
// It provides a fluent API for constructing JSON Schema objects.
type Builder interface {
	// Build serializes the schema to json.RawMessage.
	// Returns an error if the schema is invalid.
	Build() (json.RawMessage, error)

	// MustBuild is like Build but panics on error.
	MustBuild() json.RawMessage

	// schema returns the internal representation for composition.
	schema() *schemaNode
}

// schemaNode is the internal representation of a JSON Schema.
type schemaNode struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Enum        []any  `json:"enum,omitempty"`
	Default     any    `json:"default,omitempty"`

	// String constraints
	MinLength *int   `json:"minLength,omitempty"`
	MaxLength *int   `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`

	// Numeric constraints
	Minimum          *float64 `json:"minimum,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"`
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty"`

	// Array constraints
	Items       *schemaNode `json:"items,omitempty"`
	MinItems    *int        `json:"minItems,omitempty"`
	MaxItems    *int        `json:"maxItems,omitempty"`
	UniqueItems bool        `json:"uniqueItems,omitempty"`

	// Object constraints
	Properties           map[string]*schemaNode `json:"properties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	AdditionalProperties *bool                  `json:"additionalProperties,omitempty"`
}

// Sentinel errors for schema validation.
var (
	// ErrInvalidRange is returned when min exceeds max.
	ErrInvalidRange = errors.New("schema: minimum exceeds maximum")

	// ErrInvalidPattern is returned when a regex pattern is invalid.
	ErrInvalidPattern = errors.New("schema: invalid regex pattern")

	// ErrNilItems is returned when an array has no items schema.
	ErrNilItems = errors.New("schema: array requires items schema")
)

// ValidationError represents a schema validation failure.
type ValidationError struct {
	Field   string // The field name (for objects)
	Message string // Human-readable error message
	Err     error  // Underlying error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("schema: field %q: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("schema: %s", e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// validate checks the schema for internal consistency.
func (s *schemaNode) validate() error {
	switch s.Type {
	case "string":
		if s.MinLength != nil && s.MaxLength != nil && *s.MinLength > *s.MaxLength {
			return &ValidationError{
				Message: "minLength exceeds maxLength",
				Err:     ErrInvalidRange,
			}
		}
		if s.Pattern != "" {
			if _, err := regexp.Compile(s.Pattern); err != nil {
				return &ValidationError{
					Message: fmt.Sprintf("invalid pattern %q: %v", s.Pattern, err),
					Err:     ErrInvalidPattern,
				}
			}
		}

	case "integer", "number":
		if s.Minimum != nil && s.Maximum != nil && *s.Minimum > *s.Maximum {
			return &ValidationError{
				Message: "minimum exceeds maximum",
				Err:     ErrInvalidRange,
			}
		}
		if s.ExclusiveMinimum != nil && s.ExclusiveMaximum != nil && *s.ExclusiveMinimum >= *s.ExclusiveMaximum {
			return &ValidationError{
				Message: "exclusiveMinimum >= exclusiveMaximum",
				Err:     ErrInvalidRange,
			}
		}

	case "array":
		if s.Items == nil {
			return &ValidationError{
				Message: "array requires items schema",
				Err:     ErrNilItems,
			}
		}
		if s.MinItems != nil && s.MaxItems != nil && *s.MinItems > *s.MaxItems {
			return &ValidationError{
				Message: "minItems exceeds maxItems",
				Err:     ErrInvalidRange,
			}
		}
		if err := s.Items.validate(); err != nil {
			return &ValidationError{
				Message: fmt.Sprintf("invalid items schema: %v", err),
				Err:     err,
			}
		}

	case "object":
		for name, prop := range s.Properties {
			if err := prop.validate(); err != nil {
				return &ValidationError{
					Field:   name,
					Message: err.Error(),
					Err:     err,
				}
			}
		}
	}
	return nil
}

// ptr returns a pointer to the value.
func ptr[T any](v T) *T {
	return &v
}
