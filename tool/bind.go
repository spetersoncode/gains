package tool

import (
	"context"
	"encoding/json"

	ai "github.com/spetersoncode/gains"
)

// Bind creates a Tool and Handler from a typed function.
// The JSON schema for tool parameters is automatically generated
// from struct tags on type T.
//
// Example:
//
//	type TranslateArgs struct {
//	    Text string `json:"text" desc:"Text to translate" required:"true"`
//	    From string `json:"from" desc:"Source language" required:"true"`
//	    To   string `json:"to" desc:"Target language" required:"true"`
//	}
//
//	t, h := tool.Bind("translate", "Translate text between languages",
//	    func(ctx context.Context, args TranslateArgs) (string, error) {
//	        // implementation
//	        return translated, nil
//	    })
func Bind[T any](name, description string, fn TypedHandler[T]) (ai.Tool, Handler, error) {
	schema, err := SchemaFor[T]()
	if err != nil {
		return ai.Tool{}, nil, err
	}

	t := ai.Tool{
		Name:        name,
		Description: description,
		Parameters:  schema,
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args T
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", err
		}
		return fn(ctx, args)
	}

	return t, handler, nil
}

// MustBind is like Bind but panics on error.
// This is useful for initialization code where errors should be fatal.
func MustBind[T any](name, description string, fn TypedHandler[T]) (ai.Tool, Handler) {
	t, h, err := Bind(name, description, fn)
	if err != nil {
		panic(err)
	}
	return t, h
}

// BindTo creates a tool from a typed function and registers it directly to a Registry.
// This is a convenience function combining Bind and Registry.Register.
//
// Example:
//
//	type SearchArgs struct {
//	    Query string `json:"query" desc:"Search query" required:"true"`
//	    Limit int    `json:"limit" desc:"Max results" min:"1" max:"100"`
//	}
//
//	err := tool.BindTo(registry, "search", "Search the web",
//	    func(ctx context.Context, args SearchArgs) (string, error) {
//	        return doSearch(args.Query, args.Limit), nil
//	    })
func BindTo[T any](r *Registry, name, description string, fn TypedHandler[T]) error {
	t, h, err := Bind(name, description, fn)
	if err != nil {
		return err
	}
	return r.Register(t, h)
}

// MustBindTo is like BindTo but panics on error.
func MustBindTo[T any](r *Registry, name, description string, fn TypedHandler[T]) {
	if err := BindTo(r, name, description, fn); err != nil {
		panic(err)
	}
}
