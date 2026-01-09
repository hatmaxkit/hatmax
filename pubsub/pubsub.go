// Package pubsub provides a generic Pub/Sub abstraction with backend-agnostic interfaces.
//
// The package prioritizes fan-out semantics over work-queue semantics.
// Each subscriber to a topic receives its own copy of every message.
//
// Delivery is at-least-once by default. Ordering guarantees are backend-dependent
// and MUST NOT be assumed by consumers.
package pubsub

import (
	"context"
	"time"
)

// Envelope represents a transport-level message.
// Payload is application-defined; infrastructure MUST NOT inspect or mutate it.
type Envelope struct {
	ID        string
	Topic     string
	Timestamp time.Time
	Payload   any
	Metadata  map[string]string
}

// Handler processes received messages.
// Returning an error triggers backend-specific retry/failure handling.
type Handler func(ctx context.Context, env Envelope) error

// SubscribeOptions configures subscription behavior.
type SubscribeOptions struct {
	// SubscriberID identifies this subscriber for offset tracking.
	// If empty, a UUID is auto-generated (ephemeral subscription).
	// Named subscribers resume from their last offset after restart.
	SubscriberID string
}

// Publisher sends messages to topics.
// Publish MUST be non-blocking with respect to message processing.
// Errors only represent transport or persistence failures, not handler failures.
type Publisher interface {
	Publish(ctx context.Context, topic string, env Envelope) error
}

// Subscriber receives messages from topics.
// Multiple subscribers to the same topic each receive their own copy (fan-out).
// The subscriber MUST NOT assume single-consumer delivery.
type Subscriber interface {
	Subscribe(ctx context.Context, topic string, handler Handler, opts SubscribeOptions) error
}

// Broker combines Publisher and Subscriber with lifecycle management.
// Applications SHOULD depend on Publisher/Subscriber interfaces, not Broker.
type Broker interface {
	Publisher
	Subscriber
	Close() error
}
