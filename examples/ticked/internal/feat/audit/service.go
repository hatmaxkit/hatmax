package audit

import (
	"context"
	"fmt"

	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/pubsub"
)

const (
	// Topic is the pubsub topic for audit events.
	Topic = "audit.todo"
	// SubscriberID is the persistent subscriber ID for audit.
	SubscriberID = "audit-persistence"
)

// Service subscribes to audit events and persists them.
type Service struct {
	broker pubsub.Subscriber
	store  Store
	log    log.Logger
}

// NewService creates a new audit service.
func NewService(broker pubsub.Subscriber, store Store, log log.Logger) *Service {
	return &Service{
		broker: broker,
		store:  store,
		log:    log.With("component", "audit"),
	}
}

// Start subscribes to the audit topic.
func (s *Service) Start(ctx context.Context) error {
	err := s.broker.Subscribe(ctx, Topic, s.handleEvent, pubsub.SubscribeOptions{
		SubscriberID: SubscriberID,
	})
	if err != nil {
		return fmt.Errorf("cannot subscribe to %s: %w", Topic, err)
	}

	s.log.Infof("Subscribed to %s", Topic)

	return nil
}

func (s *Service) handleEvent(ctx context.Context, env pubsub.Envelope) error {
	// JSON unmarshal produces map[string]interface{}, convert to map[string]string
	rawPayload, ok := env.Payload.(map[string]interface{})
	if !ok {
		s.log.Errorf("invalid payload type: %T", env.Payload)

		return fmt.Errorf("invalid payload type: %T", env.Payload)
	}

	payload := make(map[string]string)

	for k, v := range rawPayload {
		if str, ok := v.(string); ok {
			payload[k] = str
		}
	}

	record := &Record{
		ID:        env.ID,
		EventType: payload["event_type"],
		ItemID:    payload["item_id"],
		UserID:    env.Metadata["user_id"],
		Payload:   payload,
		Source:    env.Metadata["source"],
		CreatedAt: env.Timestamp,
	}

	err := s.store.Save(ctx, record)
	if err != nil {
		s.log.Errorf("cannot save audit record: %v", err)

		return err
	}

	s.log.Debugf("Persisted audit event %s: %s", env.ID, record.EventType)

	return nil
}
