package agent

import (
	"errors"
)

// Sentinel errors for agent termination conditions.
var (
	// ErrMaxStepsReached indicates the agent hit the step limit.
	ErrMaxStepsReached = errors.New("agent: maximum steps reached")

	// ErrAgentTimeout indicates the overall timeout was exceeded.
	ErrAgentTimeout = errors.New("agent: timeout exceeded")
)
