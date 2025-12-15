package agent

import (
	"errors"
	"fmt"
)

// Sentinel errors for agent termination conditions.
var (
	// ErrMaxStepsReached indicates the agent hit the step limit.
	ErrMaxStepsReached = errors.New("agent: maximum steps reached")

	// ErrAgentTimeout indicates the overall timeout was exceeded.
	ErrAgentTimeout = errors.New("agent: timeout exceeded")
)

// ErrToolNotFound is returned when a tool call references an unregistered tool.
type ErrToolNotFound struct {
	Name string
}

func (e *ErrToolNotFound) Error() string {
	return fmt.Sprintf("agent: tool not found: %s", e.Name)
}

// ErrToolExecution wraps errors from tool handler execution.
type ErrToolExecution struct {
	Name string
	Err  error
}

func (e *ErrToolExecution) Error() string {
	return fmt.Sprintf("agent: tool %s execution failed: %v", e.Name, e.Err)
}

func (e *ErrToolExecution) Unwrap() error {
	return e.Err
}

// ErrToolAlreadyRegistered is returned when registering a tool with a duplicate name.
type ErrToolAlreadyRegistered struct {
	Name string
}

func (e *ErrToolAlreadyRegistered) Error() string {
	return fmt.Sprintf("agent: tool already registered: %s", e.Name)
}
