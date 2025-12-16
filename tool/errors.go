package tool

import "fmt"

// ErrToolNotFound is returned when a tool call references an unregistered tool.
type ErrToolNotFound struct {
	Name string
}

// Error returns a formatted error message including the tool name.
func (e *ErrToolNotFound) Error() string {
	return fmt.Sprintf("tool: not found: %s", e.Name)
}

// ErrToolExecution wraps errors from tool handler execution.
type ErrToolExecution struct {
	Name string
	Err  error
}

// Error returns a formatted error message including the tool name and cause.
func (e *ErrToolExecution) Error() string {
	return fmt.Sprintf("tool: %s execution failed: %v", e.Name, e.Err)
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *ErrToolExecution) Unwrap() error {
	return e.Err
}

// ErrToolAlreadyRegistered is returned when registering a tool with a duplicate name.
type ErrToolAlreadyRegistered struct {
	Name string
}

// Error returns a formatted error message including the duplicate tool name.
func (e *ErrToolAlreadyRegistered) Error() string {
	return fmt.Sprintf("tool: already registered: %s", e.Name)
}
