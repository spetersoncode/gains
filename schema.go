package gains

import (
	"encoding/json"
	"reflect"
	"strings"
)

// SchemaBuilder provides a fluent API for constructing JSON Schema objects
// from Go structs. Use SchemaFrom[T]() to create a builder from a struct type.
type SchemaBuilder struct {
	properties    map[string]*propertyDef
	required      []string
	propertyOrder []string
}

// propertyDef holds the definition of a single property.
type propertyDef struct {
	Type        string         `json:"type"`
	Description string         `json:"description,omitempty"`
	Enum        []any          `json:"enum,omitempty"`
	Items       *propertyDef   `json:"items,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
	Required    []string       `json:"required,omitempty"`
}

// SchemaFrom creates a SchemaBuilder by reflecting on the given struct type.
// Field names are taken from json tags, types are mapped to JSON Schema types.
func SchemaFrom[T any]() *SchemaBuilder {
	var zero T
	t := reflect.TypeOf(zero)

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return &SchemaBuilder{
			properties:    make(map[string]*propertyDef),
			propertyOrder: make([]string, 0),
		}
	}

	return buildFromStruct(t)
}

func buildFromStruct(t reflect.Type) *SchemaBuilder {
	sb := &SchemaBuilder{
		properties:    make(map[string]*propertyDef),
		propertyOrder: make([]string, 0),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON field name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		name := strings.Split(jsonTag, ",")[0]
		if name == "" {
			name = field.Name
		}

		prop := typeToPropertyDef(field.Type)
		sb.properties[name] = prop
		sb.propertyOrder = append(sb.propertyOrder, name)
	}

	return sb
}

func typeToPropertyDef(t reflect.Type) *propertyDef {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return &propertyDef{Type: "string"}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &propertyDef{Type: "integer"}

	case reflect.Float32, reflect.Float64:
		return &propertyDef{Type: "number"}

	case reflect.Bool:
		return &propertyDef{Type: "boolean"}

	case reflect.Slice, reflect.Array:
		items := typeToPropertyDef(t.Elem())
		return &propertyDef{Type: "array", Items: items}

	case reflect.Struct:
		nested := buildFromStruct(t)
		props := make(map[string]any)
		for _, name := range nested.propertyOrder {
			props[name] = nested.properties[name].toMap()
		}
		prop := &propertyDef{Type: "object", Properties: props}
		if len(nested.required) > 0 {
			prop.Required = nested.required
		}
		return prop

	case reflect.Map:
		// Maps become objects with no defined properties
		return &propertyDef{Type: "object"}

	default:
		return &propertyDef{Type: "string"}
	}
}

// Desc sets the description for a field.
func (s *SchemaBuilder) Desc(field, description string) *SchemaBuilder {
	if prop, ok := s.properties[field]; ok {
		prop.Description = description
	}
	return s
}

// Required marks the specified fields as required.
func (s *SchemaBuilder) Required(fields ...string) *SchemaBuilder {
	for _, field := range fields {
		if _, ok := s.properties[field]; ok {
			// Avoid duplicates
			found := false
			for _, r := range s.required {
				if r == field {
					found = true
					break
				}
			}
			if !found {
				s.required = append(s.required, field)
			}
		}
	}
	return s
}

// Enum sets the allowed values for a string field.
func (s *SchemaBuilder) Enum(field string, values ...string) *SchemaBuilder {
	if prop, ok := s.properties[field]; ok {
		prop.Enum = make([]any, len(values))
		for i, v := range values {
			prop.Enum[i] = v
		}
	}
	return s
}

// Build generates the JSON Schema as json.RawMessage.
func (s *SchemaBuilder) Build() json.RawMessage {
	schema := s.toMap()
	data, err := json.Marshal(schema)
	if err != nil {
		// This should never happen with valid Go types
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	return data
}

func (s *SchemaBuilder) toMap() map[string]any {
	result := map[string]any{
		"type": "object",
	}

	if len(s.properties) > 0 {
		props := make(map[string]any)
		for _, name := range s.propertyOrder {
			props[name] = s.properties[name].toMap()
		}
		result["properties"] = props
	} else {
		result["properties"] = map[string]any{}
	}

	if len(s.required) > 0 {
		result["required"] = s.required
	}

	return result
}

func (p *propertyDef) toMap() map[string]any {
	result := map[string]any{
		"type": p.Type,
	}

	if p.Description != "" {
		result["description"] = p.Description
	}

	if len(p.Enum) > 0 {
		result["enum"] = p.Enum
	}

	if p.Items != nil {
		result["items"] = p.Items.toMap()
	}

	if p.Properties != nil {
		result["properties"] = p.Properties
	}

	if len(p.Required) > 0 {
		result["required"] = p.Required
	}

	return result
}
