package observer

import (
	"context"

	"github.com/spetersoncode/gains/event"
)

// Noop is an observer that discards all events.
// It is the default observer when none is configured.
type Noop struct{}

// NewNoop creates a new no-op observer.
func NewNoop() *Noop {
	return &Noop{}
}

// Observe discards the event.
func (n *Noop) Observe(ctx context.Context, e event.Event) {}

// Close is a no-op for the Noop observer.
func (n *Noop) Close() error { return nil }

// Verify Noop implements Observer.
var _ Observer = (*Noop)(nil)
