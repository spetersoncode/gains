package tool

import (
	"context"
	"encoding/json"
	"testing"

	ai "github.com/spetersoncode/gains"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testArgs struct {
	Query string `json:"query" desc:"Search query" required:"true"`
}

type calcArgs struct {
	A int `json:"a" required:"true"`
	B int `json:"b" required:"true"`
}

func TestRegistryAdd(t *testing.T) {
	t.Run("registers single tool with Func", func(t *testing.T) {
		registry := NewRegistry().Add(
			Func("search", "Search the web", func(ctx context.Context, args testArgs) (string, error) {
				return "result: " + args.Query, nil
			}),
		)

		assert.Equal(t, 1, registry.Len())
		handler, ok := registry.Get("search")
		assert.True(t, ok)
		assert.NotNil(t, handler)

		tool, ok := registry.GetTool("search")
		assert.True(t, ok)
		assert.Equal(t, "search", tool.Name)
		assert.Equal(t, "Search the web", tool.Description)
	})

	t.Run("registers multiple tools in single Add call", func(t *testing.T) {
		registry := NewRegistry().Add(
			Func("search", "Search the web", func(ctx context.Context, args testArgs) (string, error) {
				return "search result", nil
			}),
			Func("calc", "Calculate sum", func(ctx context.Context, args calcArgs) (string, error) {
				return "calc result", nil
			}),
		)

		assert.Equal(t, 2, registry.Len())
		assert.Contains(t, registry.Names(), "search")
		assert.Contains(t, registry.Names(), "calc")
	})

	t.Run("chains multiple Add calls", func(t *testing.T) {
		registry := NewRegistry().
			Add(Func("first", "First tool", func(ctx context.Context, args testArgs) (string, error) {
				return "first", nil
			})).
			Add(Func("second", "Second tool", func(ctx context.Context, args testArgs) (string, error) {
				return "second", nil
			})).
			Add(Func("third", "Third tool", func(ctx context.Context, args testArgs) (string, error) {
				return "third", nil
			}))

		assert.Equal(t, 3, registry.Len())
		assert.Contains(t, registry.Names(), "first")
		assert.Contains(t, registry.Names(), "second")
		assert.Contains(t, registry.Names(), "third")
	})

	t.Run("panics on duplicate tool name", func(t *testing.T) {
		assert.Panics(t, func() {
			NewRegistry().Add(
				Func("dupe", "First", func(ctx context.Context, args testArgs) (string, error) {
					return "", nil
				}),
				Func("dupe", "Duplicate", func(ctx context.Context, args testArgs) (string, error) {
					return "", nil
				}),
			)
		})
	})
}

func TestFunc(t *testing.T) {
	t.Run("creates Registration with correct tool definition", func(t *testing.T) {
		reg := Func("myTool", "My description", func(ctx context.Context, args testArgs) (string, error) {
			return args.Query, nil
		})

		assert.Equal(t, "myTool", reg.Tool.Name)
		assert.Equal(t, "My description", reg.Tool.Description)
		assert.NotNil(t, reg.Tool.Parameters)
		assert.NotNil(t, reg.Handler)
	})

	t.Run("handler correctly unmarshals arguments", func(t *testing.T) {
		reg := Func("test", "Test", func(ctx context.Context, args testArgs) (string, error) {
			return "got: " + args.Query, nil
		})

		result, err := reg.Handler(context.Background(), ai.ToolCall{
			ID:        "call_1",
			Name:      "test",
			Arguments: `{"query": "hello world"}`,
		})

		require.NoError(t, err)
		assert.Equal(t, "got: hello world", result)
	})

	t.Run("handler returns error on invalid JSON", func(t *testing.T) {
		reg := Func("test", "Test", func(ctx context.Context, args testArgs) (string, error) {
			return args.Query, nil
		})

		_, err := reg.Handler(context.Background(), ai.ToolCall{
			ID:        "call_1",
			Name:      "test",
			Arguments: `{invalid json}`,
		})

		assert.Error(t, err)
	})
}

func TestWithHandler(t *testing.T) {
	t.Run("creates Registration from Handler", func(t *testing.T) {
		schema := json.RawMessage(`{"type": "object"}`)
		handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
			return "handled", nil
		}

		reg := WithHandler("custom", "Custom handler", schema, handler)

		assert.Equal(t, "custom", reg.Tool.Name)
		assert.Equal(t, "Custom handler", reg.Tool.Description)
		assert.Equal(t, schema, reg.Tool.Parameters)
		assert.NotNil(t, reg.Handler)
	})
}

func TestWithTool(t *testing.T) {
	t.Run("creates Registration from existing Tool", func(t *testing.T) {
		tool := ai.Tool{
			Name:        "existing",
			Description: "Existing tool",
			Parameters:  json.RawMessage(`{"type": "object"}`),
		}
		handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
			return "handled", nil
		}

		reg := WithTool(tool, handler)

		assert.Equal(t, tool.Name, reg.Tool.Name)
		assert.Equal(t, tool.Description, reg.Tool.Description)
		assert.Equal(t, tool.Parameters, reg.Tool.Parameters)
		assert.NotNil(t, reg.Handler)
	})
}

func TestRegistryExecuteWithFluentRegistration(t *testing.T) {
	t.Run("executes tool registered via fluent API", func(t *testing.T) {
		registry := NewRegistry().Add(
			Func("greet", "Greet someone", func(ctx context.Context, args struct {
				Name string `json:"name" required:"true"`
			}) (string, error) {
				return "Hello, " + args.Name + "!", nil
			}),
		)

		result, err := registry.Execute(context.Background(), ai.ToolCall{
			ID:        "call_123",
			Name:      "greet",
			Arguments: `{"name": "World"}`,
		})

		require.NoError(t, err)
		assert.Equal(t, "call_123", result.ToolCallID)
		assert.Equal(t, "Hello, World!", result.Content)
		assert.False(t, result.IsError)
	})
}
