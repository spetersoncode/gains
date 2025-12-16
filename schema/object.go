package schema

import (
	"encoding/json"
	"fmt"
)

// Object creates a new object schema builder.
func Object() *ObjectBuilder {
	return &ObjectBuilder{
		node: &schemaNode{
			Type:       "object",
			Properties: make(map[string]*schemaNode),
		},
	}
}

// ObjectBuilder constructs object type schemas.
type ObjectBuilder struct {
	node *schemaNode
}

// Desc sets the description for the object itself.
func (b *ObjectBuilder) Desc(description string) *ObjectBuilder {
	b.node.Description = description
	return b
}

// Field adds a field with its schema.
// The field argument can be a Builder or a *RequiredField.
func (b *ObjectBuilder) Field(name string, field any) *ObjectBuilder {
	switch f := field.(type) {
	case *RequiredField:
		b.node.Properties[name] = f.builder.schema()
		b.addRequired(name)
	case Builder:
		b.node.Properties[name] = f.schema()
	default:
		panic(fmt.Sprintf("schema: Field %q requires a Builder or *RequiredField, got %T", name, field))
	}
	return b
}

// addRequired adds a field to the required list without duplicates.
func (b *ObjectBuilder) addRequired(name string) {
	for _, r := range b.node.Required {
		if r == name {
			return
		}
	}
	b.node.Required = append(b.node.Required, name)
}

// AdditionalProperties controls whether extra properties are allowed.
// OpenAI strict mode requires this to be false.
func (b *ObjectBuilder) AdditionalProperties(allowed bool) *ObjectBuilder {
	b.node.AdditionalProperties = ptr(allowed)
	return b
}

// StrictMode sets additionalProperties to false for OpenAI compatibility.
func (b *ObjectBuilder) StrictMode() *ObjectBuilder {
	return b.AdditionalProperties(false)
}

// Required marks this object as required when nested in another object.
func (b *ObjectBuilder) Required() *RequiredField {
	return &RequiredField{builder: b}
}

// Build serializes the schema to json.RawMessage.
func (b *ObjectBuilder) Build() (json.RawMessage, error) {
	if err := b.node.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(b.node)
}

// MustBuild is like Build but panics on error.
func (b *ObjectBuilder) MustBuild() json.RawMessage {
	data, err := b.Build()
	if err != nil {
		panic(err)
	}
	return data
}

func (b *ObjectBuilder) schema() *schemaNode {
	return b.node
}

// RequiredField wraps a Builder to mark it as required in an object.
type RequiredField struct {
	builder Builder
}
