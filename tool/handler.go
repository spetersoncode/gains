package tool

import (
	"context"

	ai "github.com/spetersoncode/gains"
)

// Handler is a function that executes a tool call and returns a result.
// The context supports cancellation and timeout.
// The call contains the tool name, ID, and arguments as a JSON string.
// Returns the result content string, or an error if execution failed.
type Handler func(ctx context.Context, call ai.ToolCall) (string, error)

// TypedHandler is a function that executes a tool call with typed arguments.
// The args parameter is automatically unmarshaled from the tool call's JSON arguments.
type TypedHandler[T any] func(ctx context.Context, args T) (string, error)
