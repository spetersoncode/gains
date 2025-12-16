package tool

import (
	"github.com/spetersoncode/gains/client"
)

// RegisterAll registers all tool pairs to a registry.
// Returns the first error encountered, or nil if all registrations succeed.
func RegisterAll(r *Registry, pairs []ToolPair) error {
	for _, p := range pairs {
		if err := r.Register(p.Tool, p.Handler); err != nil {
			return err
		}
	}
	return nil
}

// MustRegisterAll is like RegisterAll but panics on error.
func MustRegisterAll(r *Registry, pairs []ToolPair) {
	if err := RegisterAll(r, pairs); err != nil {
		panic(err)
	}
}

// AllToolsOption configures the AllTools function.
type AllToolsOption func(*allToolsConfig)

type allToolsConfig struct {
	fileOpts   []FileToolOption
	httpOpts   []HTTPToolOption
	searchOpts []SearchToolOption
	clientOpts []ClientToolsOption
}

// WithFileOptions sets options for file tools in AllTools.
func WithFileOptions(opts ...FileToolOption) AllToolsOption {
	return func(c *allToolsConfig) {
		c.fileOpts = opts
	}
}

// WithHTTPOptions sets options for HTTP tool in AllTools.
func WithHTTPOptions(opts ...HTTPToolOption) AllToolsOption {
	return func(c *allToolsConfig) {
		c.httpOpts = opts
	}
}

// WithSearchOptions sets options for search tool in AllTools.
func WithSearchOptions(opts ...SearchToolOption) AllToolsOption {
	return func(c *allToolsConfig) {
		c.searchOpts = opts
	}
}

// WithClientOptions sets options for client tools in AllTools.
func WithClientOptions(opts ...ClientToolsOption) AllToolsOption {
	return func(c *allToolsConfig) {
		c.clientOpts = opts
	}
}

// AllTools returns all standard tools: file, HTTP, search, and client tools.
// Pass nil for the client parameter to exclude client-specific tools.
func AllTools(c *client.Client, opts ...AllToolsOption) []ToolPair {
	cfg := &allToolsConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var pairs []ToolPair

	// Add file tools
	pairs = append(pairs, FileTools(cfg.fileOpts...)...)

	// Add HTTP tool
	pairs = append(pairs, WebTools(cfg.httpOpts...)...)

	// Add search tool
	pairs = append(pairs, SearchTools(cfg.searchOpts...)...)

	// Add client tools if client is provided
	if c != nil {
		pairs = append(pairs, ClientTools(c, cfg.clientOpts...)...)
	}

	return pairs
}

// StandardTools returns non-client tools: file, HTTP, and search tools.
// Use this when you don't need image, embedding, or chat tools.
func StandardTools(opts ...AllToolsOption) []ToolPair {
	cfg := &allToolsConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var pairs []ToolPair

	// Add file tools
	pairs = append(pairs, FileTools(cfg.fileOpts...)...)

	// Add HTTP tool
	pairs = append(pairs, WebTools(cfg.httpOpts...)...)

	// Add search tool
	pairs = append(pairs, SearchTools(cfg.searchOpts...)...)

	return pairs
}
