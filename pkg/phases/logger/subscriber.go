package logger

import (
	"context"

	"github.com/get-glu/glu/pkg/core"
	"github.com/google/uuid"
)

type Event struct {
	Phase core.Descriptor
	State core.State
	// TODO: event type
}

// Subscriber is a type that can receive events from a phase logger.
type Subscriber interface {
	// OnEvent is called when a new event is received.
	OnEvent(ctx context.Context, event Event) error
}

// Subscription represents an active subscription to a phase logger.
type Subscription struct {
	ID     uuid.UUID
	Cancel context.CancelFunc
}

// Subscribable extends PhaseLogger with subscription functionality.
type Subscribable[R core.Resource] interface {
	PhaseLogger[R]

	// Subscribe returns a new subscription to the phase logger.
	Subscribe(ctx context.Context, subscriber Subscriber) (*Subscription, error)
	// Unsubscribe cancels a subscription.
	Unsubscribe(ctx context.Context, id uuid.UUID) error
}
