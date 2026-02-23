package pubsub

import (
	"context"
	"sync"
)

// NoopBroker is a no-operation broker for testing.
// It captures published messages for assertions but does not deliver them.
type NoopBroker struct {
	mu        sync.Mutex
	published []Envelope
}

// NewNoopBroker creates a new NoopBroker.
func NewNoopBroker() *NoopBroker {
	return &NoopBroker{
		published: make([]Envelope, 0),
	}
}

// Publish captures the envelope for later inspection.
func (b *NoopBroker) Publish(ctx context.Context, topic string, env Envelope) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.published = append(b.published, env)

	return nil
}

// Subscribe is a no-op that returns immediately.
func (b *NoopBroker) Subscribe(ctx context.Context, topic string, handler Handler, opts SubscribeOptions) error {
	return nil
}

// Close is a no-op.
func (b *NoopBroker) Close() error {
	return nil
}

// Published returns all captured envelopes.
func (b *NoopBroker) Published() []Envelope {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := make([]Envelope, len(b.published))
	copy(result, b.published)

	return result
}

// Reset clears all captured envelopes.
func (b *NoopBroker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.published = make([]Envelope, 0)
}

// MemoryBroker is an in-memory broker for integration tests.
// It delivers messages synchronously to all subscribers (fan-out).
type MemoryBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]Handler
}

// NewMemoryBroker creates a new MemoryBroker.
func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{
		subscribers: make(map[string][]Handler),
	}
}

// Publish delivers the envelope to all subscribers of the topic synchronously.
func (b *MemoryBroker) Publish(ctx context.Context, topic string, env Envelope) error {
	b.mu.RLock()
	handlers := b.subscribers[topic]
	b.mu.RUnlock()

	for _, handler := range handlers {
		err := handler(ctx, env)
		if err != nil {
			// Continue delivery to other handlers even if one fails
			continue
		}
	}

	return nil
}

// Subscribe registers a handler for the topic.
func (b *MemoryBroker) Subscribe(ctx context.Context, topic string, handler Handler, opts SubscribeOptions) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[topic] = append(b.subscribers[topic], handler)

	return nil
}

// Close clears all subscriptions.
func (b *MemoryBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers = make(map[string][]Handler)

	return nil
}
