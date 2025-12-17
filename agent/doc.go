// Package agent provides autonomous tool-calling agent functionality for the gains library.
//
// An agent orchestrates a conversation loop where the model can request tool calls,
// which are automatically executed and the results fed back to the model until
// the model produces a final response without tool calls.
//
// # Basic Usage
//
// Create a registry, register tools with their handlers, then create an agent:
//
//	// Define tool arguments
//	type WeatherArgs struct {
//	    Location string `json:"location" desc:"City name" required:"true"`
//	}
//
//	// Create registry and register tool
//	registry := tool.NewRegistry()
//	tool.MustRegisterFunc(registry, "get_weather", "Get current weather",
//	    func(ctx context.Context, args WeatherArgs) (string, error) {
//	        return fmt.Sprintf(`{"temp": 72, "location": %q}`, args.Location), nil
//	    },
//	)
//
//	// Create agent
//	a := agent.New(client, registry)
//
//	// Run and get final result (blocking)
//	result, err := a.Run(ctx, messages, agent.WithMaxSteps(5))
//
// # Streaming Events
//
// Use RunStream() to receive events as the agent executes:
//
//	import "github.com/spetersoncode/gains/event"
//
//	events := a.RunStream(ctx, messages, agent.WithMaxSteps(5))
//	for e := range events {
//	    switch e.Type {
//	    case event.MessageDelta:
//	        fmt.Print(e.Delta)
//	    case event.ToolCallStart:
//	        fmt.Printf("[Tool: %s]\n", e.ToolCall.Name)
//	    case event.RunEnd:
//	        fmt.Println("Done!")
//	    }
//	}
//
// # Human-in-the-Loop Approval
//
// Use WithApprover to require approval before tool execution:
//
//	events := a.Run(ctx, messages,
//	    agent.WithApprover(func(ctx context.Context, call gains.ToolCall) (bool, string) {
//	        fmt.Printf("Approve %s? (y/n): ", call.Name)
//	        var input string
//	        fmt.Scanln(&input)
//	        return input == "y", "User rejected"
//	    }),
//	)
//
// # Configuration Options
//
// The agent supports various configuration options:
//
//   - WithMaxSteps(n): Limit iterations to prevent infinite loops (default: 10)
//   - WithTimeout(d): Set overall execution timeout
//   - WithHandlerTimeout(d): Set per-handler timeout (default: 30s)
//   - WithParallelToolCalls(bool): Enable/disable parallel tool execution (default: true)
//   - WithApprover(fn): Enable human-in-the-loop approval
//   - WithApprovalRequired(tools...): Require approval only for specific tools
//   - WithStopPredicate(fn): Custom termination condition
//   - WithChatOptions(opts...): Pass options to underlying ChatProvider
//
// # Termination Conditions
//
// The agent stops when any of these conditions are met:
//
//   - The model responds without tool calls (TerminationComplete)
//   - MaxSteps is reached (TerminationMaxSteps)
//   - Timeout is exceeded (TerminationTimeout)
//   - Context is cancelled (TerminationCancelled)
//   - StopPredicate returns true (TerminationCustom)
//   - All tool calls are rejected (TerminationRejected)
//   - An error occurs (TerminationError)
package agent
