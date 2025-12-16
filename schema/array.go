package schema

import "encoding/json"

// Array creates a new array schema builder with the specified item type.
func Array(items Builder) *ArrayBuilder {
	return &ArrayBuilder{
		node: &schemaNode{
			Type:  "array",
			Items: items.schema(),
		},
	}
}

// ArrayBuilder constructs array type schemas.
type ArrayBuilder struct {
	node *schemaNode
}

// Desc sets the description.
func (b *ArrayBuilder) Desc(description string) *ArrayBuilder {
	b.node.Description = description
	return b
}

// MinItems sets the minimum number of items.
func (b *ArrayBuilder) MinItems(n int) *ArrayBuilder {
	b.node.MinItems = ptr(n)
	return b
}

// MaxItems sets the maximum number of items.
func (b *ArrayBuilder) MaxItems(n int) *ArrayBuilder {
	b.node.MaxItems = ptr(n)
	return b
}

// UniqueItems requires all items to be unique.
func (b *ArrayBuilder) UniqueItems() *ArrayBuilder {
	b.node.UniqueItems = true
	return b
}

// Required marks this field as required when used in an object.
func (b *ArrayBuilder) Required() *RequiredField {
	return &RequiredField{builder: b}
}

// Build serializes the schema to json.RawMessage.
func (b *ArrayBuilder) Build() (json.RawMessage, error) {
	if err := b.node.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(b.node)
}

// MustBuild is like Build but panics on error.
func (b *ArrayBuilder) MustBuild() json.RawMessage {
	data, err := b.Build()
	if err != nil {
		panic(err)
	}
	return data
}

func (b *ArrayBuilder) schema() *schemaNode {
	return b.node
}
