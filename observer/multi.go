package observer

import (
	"context"
	"errors"

	"github.com/spetersoncode/gains/event"
)

// Multi fans out events to multiple observers.
// All observers receive every event. Errors from Close are combined.
type Multi struct {
	observers []Observer
}

// NewMulti creates a new multi-observer that forwards events to all given observers.
// If no observers are provided, it behaves like a Noop observer.
func NewMulti(observers ...Observer) *Multi {
	return &Multi{observers: observers}
}

// Observe forwards the event to all observers.
// Events are delivered synchronously to each observer in order.
func (m *Multi) Observe(ctx context.Context, e event.Event) {
	for _, obs := range m.observers {
		obs.Observe(ctx, e)
	}
}

// Close closes all observers, combining any errors.
func (m *Multi) Close() error {
	var errs []error
	for _, obs := range m.observers {
		if err := obs.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Add appends an observer to the multi-observer.
// This is not safe for concurrent use with Observe.
func (m *Multi) Add(obs Observer) {
	m.observers = append(m.observers, obs)
}

// Verify Multi implements Observer.
var _ Observer = (*Multi)(nil)
