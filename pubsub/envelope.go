package pubsub

import (
	"time"

	"github.com/google/uuid"
)

// NewEnvelope creates a new Envelope with auto-generated ID and current timestamp.
func NewEnvelope(topic string, payload any) Envelope {
	return Envelope{
		ID:        uuid.New().String(),
		Topic:     topic,
		Timestamp: time.Now(),
		Payload:   payload,
		Metadata:  make(map[string]string),
	}
}

// NewEnvelopeWithMetadata creates a new Envelope with the provided metadata.
func NewEnvelopeWithMetadata(topic string, payload any, metadata map[string]string) Envelope {
	env := NewEnvelope(topic, payload)
	if metadata != nil {
		env.Metadata = metadata
	}
	return env
}

// WithMetadata returns a copy of the envelope with the key-value pair added to metadata.
func (e Envelope) WithMetadata(key, value string) Envelope {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}
