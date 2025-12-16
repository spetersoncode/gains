package schema

import "encoding/json"

// Bool creates a new boolean schema builder.
func Bool() *BoolBuilder {
	return &BoolBuilder{
		node: &schemaNode{Type: "boolean"},
	}
}

// Boolean is an alias for Bool.
func Boolean() *BoolBuilder {
	return Bool()
}

// BoolBuilder constructs boolean type schemas.
type BoolBuilder struct {
	node *schemaNode
}

// Desc sets the description.
func (b *BoolBuilder) Desc(description string) *BoolBuilder {
	b.node.Description = description
	return b
}

// Default sets the default value.
func (b *BoolBuilder) Default(value bool) *BoolBuilder {
	b.node.Default = value
	return b
}

// Required marks this field as required when used in an object.
func (b *BoolBuilder) Required() *RequiredField {
	return &RequiredField{builder: b}
}

// Build serializes the schema to json.RawMessage.
func (b *BoolBuilder) Build() (json.RawMessage, error) {
	if err := b.node.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(b.node)
}

// MustBuild is like Build but panics on error.
func (b *BoolBuilder) MustBuild() json.RawMessage {
	data, err := b.Build()
	if err != nil {
		panic(err)
	}
	return data
}

func (b *BoolBuilder) schema() *schemaNode {
	return b.node
}
