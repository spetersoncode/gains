package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spetersoncode/gains/tool"
)

// SetupDemoTools registers demo tools for testing the server.
// These tools are enabled by default (GAINS_DEMO_TOOLS=true).
func SetupDemoTools(registry *tool.Registry) {
	// Weather tool - classic demo
	tool.MustRegisterFunc(registry, "get_weather",
		"Get the current weather for a location",
		func(ctx context.Context, args struct {
			Location string `json:"location" desc:"City name, e.g. Paris" required:"true"`
		}) (string, error) {
			time.Sleep(50 * time.Millisecond) // Simulate API latency
			return fmt.Sprintf(`{"location": %q, "temperature": 22, "conditions": "Sunny", "unit": "celsius"}`, args.Location), nil
		},
	)

	// Time tool
	tool.MustRegisterFunc(registry, "get_time",
		"Get the current time",
		func(ctx context.Context, args struct{}) (string, error) {
			return fmt.Sprintf(`{"time": %q, "timezone": "UTC"}`, time.Now().UTC().Format(time.RFC3339)), nil
		},
	)

	// Echo tool - useful for testing
	tool.MustRegisterFunc(registry, "echo",
		"Echo back the input message (useful for testing)",
		func(ctx context.Context, args struct {
			Message string `json:"message" desc:"Message to echo back" required:"true"`
		}) (string, error) {
			return fmt.Sprintf(`{"echo": %q}`, args.Message), nil
		},
	)
}
