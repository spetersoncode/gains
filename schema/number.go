package schema

import "encoding/json"

// Int creates a new integer schema builder.
func Int() *IntBuilder {
	return &IntBuilder{
		node: &schemaNode{Type: "integer"},
	}
}

// Integer is an alias for Int.
func Integer() *IntBuilder {
	return Int()
}

// IntBuilder constructs integer type schemas.
type IntBuilder struct {
	node *schemaNode
}

// Desc sets the description.
func (b *IntBuilder) Desc(description string) *IntBuilder {
	b.node.Description = description
	return b
}

// Min sets the minimum value (inclusive).
func (b *IntBuilder) Min(n int) *IntBuilder {
	b.node.Minimum = ptr(float64(n))
	return b
}

// Max sets the maximum value (inclusive).
func (b *IntBuilder) Max(n int) *IntBuilder {
	b.node.Maximum = ptr(float64(n))
	return b
}

// ExclusiveMin sets the exclusive minimum value.
func (b *IntBuilder) ExclusiveMin(n int) *IntBuilder {
	b.node.ExclusiveMinimum = ptr(float64(n))
	return b
}

// ExclusiveMax sets the exclusive maximum value.
func (b *IntBuilder) ExclusiveMax(n int) *IntBuilder {
	b.node.ExclusiveMaximum = ptr(float64(n))
	return b
}

// Enum restricts the value to specific integers.
func (b *IntBuilder) Enum(values ...int) *IntBuilder {
	b.node.Enum = make([]any, len(values))
	for i, v := range values {
		b.node.Enum[i] = v
	}
	return b
}

// Default sets the default value.
func (b *IntBuilder) Default(value int) *IntBuilder {
	b.node.Default = value
	return b
}

// Required marks this field as required when used in an object.
func (b *IntBuilder) Required() *RequiredField {
	return &RequiredField{builder: b}
}

// Build serializes the schema to json.RawMessage.
func (b *IntBuilder) Build() (json.RawMessage, error) {
	if err := b.node.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(b.node)
}

// MustBuild is like Build but panics on error.
func (b *IntBuilder) MustBuild() json.RawMessage {
	data, err := b.Build()
	if err != nil {
		panic(err)
	}
	return data
}

func (b *IntBuilder) schema() *schemaNode {
	return b.node
}

// Number creates a new number (float) schema builder.
func Number() *NumberBuilder {
	return &NumberBuilder{
		node: &schemaNode{Type: "number"},
	}
}

// NumberBuilder constructs number (float) type schemas.
type NumberBuilder struct {
	node *schemaNode
}

// Desc sets the description.
func (b *NumberBuilder) Desc(description string) *NumberBuilder {
	b.node.Description = description
	return b
}

// Min sets the minimum value (inclusive).
func (b *NumberBuilder) Min(n float64) *NumberBuilder {
	b.node.Minimum = ptr(n)
	return b
}

// Max sets the maximum value (inclusive).
func (b *NumberBuilder) Max(n float64) *NumberBuilder {
	b.node.Maximum = ptr(n)
	return b
}

// ExclusiveMin sets the exclusive minimum value.
func (b *NumberBuilder) ExclusiveMin(n float64) *NumberBuilder {
	b.node.ExclusiveMinimum = ptr(n)
	return b
}

// ExclusiveMax sets the exclusive maximum value.
func (b *NumberBuilder) ExclusiveMax(n float64) *NumberBuilder {
	b.node.ExclusiveMaximum = ptr(n)
	return b
}

// Enum restricts the value to specific numbers.
func (b *NumberBuilder) Enum(values ...float64) *NumberBuilder {
	b.node.Enum = make([]any, len(values))
	for i, v := range values {
		b.node.Enum[i] = v
	}
	return b
}

// Default sets the default value.
func (b *NumberBuilder) Default(value float64) *NumberBuilder {
	b.node.Default = value
	return b
}

// Required marks this field as required when used in an object.
func (b *NumberBuilder) Required() *RequiredField {
	return &RequiredField{builder: b}
}

// Build serializes the schema to json.RawMessage.
func (b *NumberBuilder) Build() (json.RawMessage, error) {
	if err := b.node.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(b.node)
}

// MustBuild is like Build but panics on error.
func (b *NumberBuilder) MustBuild() json.RawMessage {
	data, err := b.Build()
	if err != nil {
		panic(err)
	}
	return data
}

func (b *NumberBuilder) schema() *schemaNode {
	return b.node
}
