package schema

import "encoding/json"

// String creates a new string schema builder.
func String() *StringBuilder {
	return &StringBuilder{
		node: &schemaNode{Type: "string"},
	}
}

// StringBuilder constructs string type schemas.
type StringBuilder struct {
	node *schemaNode
}

// Desc sets the description for this field.
func (b *StringBuilder) Desc(description string) *StringBuilder {
	b.node.Description = description
	return b
}

// Enum restricts the value to one of the provided options.
func (b *StringBuilder) Enum(values ...string) *StringBuilder {
	b.node.Enum = make([]any, len(values))
	for i, v := range values {
		b.node.Enum[i] = v
	}
	return b
}

// MinLength sets the minimum string length.
func (b *StringBuilder) MinLength(n int) *StringBuilder {
	b.node.MinLength = ptr(n)
	return b
}

// MaxLength sets the maximum string length.
func (b *StringBuilder) MaxLength(n int) *StringBuilder {
	b.node.MaxLength = ptr(n)
	return b
}

// Pattern sets a regex pattern the string must match.
func (b *StringBuilder) Pattern(regex string) *StringBuilder {
	b.node.Pattern = regex
	return b
}

// Default sets the default value.
func (b *StringBuilder) Default(value string) *StringBuilder {
	b.node.Default = value
	return b
}

// Required marks this field as required when used in an object.
// Returns a RequiredField wrapper for use with ObjectBuilder.Field().
func (b *StringBuilder) Required() *RequiredField {
	return &RequiredField{builder: b}
}

// Build serializes the schema to json.RawMessage.
func (b *StringBuilder) Build() (json.RawMessage, error) {
	if err := b.node.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(b.node)
}

// MustBuild is like Build but panics on error.
func (b *StringBuilder) MustBuild() json.RawMessage {
	data, err := b.Build()
	if err != nil {
		panic(err)
	}
	return data
}

func (b *StringBuilder) schema() *schemaNode {
	return b.node
}
