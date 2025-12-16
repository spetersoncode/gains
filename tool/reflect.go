package tool

import (
	"encoding/json"

	ai "github.com/spetersoncode/gains"
)

// SchemaFor generates a JSON schema from a struct type T.
// This is a convenience re-export of gains.SchemaFor.
// See gains.SchemaFor for full documentation.
func SchemaFor[T any]() (json.RawMessage, error) {
	return ai.SchemaFor[T]()
}

// MustSchemaFor is like SchemaFor but panics on error.
// This is a convenience re-export of gains.MustSchemaFor.
func MustSchemaFor[T any]() json.RawMessage {
	return ai.MustSchemaFor[T]()
}
