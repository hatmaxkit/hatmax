package list

import (
	"context"
	"errors"

	"github.com/hatmaxkit/hatmax/examples/ticked/internal/feat/audit"
	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/pubsub"
)

// Service provides todo list business logic.
type Service struct {
	store     Store
	publisher pubsub.Publisher
	log       log.Logger
}

// NewService creates a new list service.
func NewService(store Store, publisher pubsub.Publisher, log log.Logger) *Service {
	return &Service{
		store:     store,
		publisher: publisher,
		log:       log,
	}
}

// GetOrCreateList retrieves a user's list or creates a new one if it doesn't exist.
func (s *Service) GetOrCreateList(ctx context.Context, userID string) (*TodoList, error) {
	list, err := s.store.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			list = NewTodoList(userID)
			if err := s.store.Save(ctx, list); err != nil {
				return nil, err
			}
			s.log.Infof("created new list for user %s", userID)
			return list, nil
		}
		return nil, err
	}
	return list, nil
}

// AddItem adds an item to a user's list.
func (s *Service) AddItem(ctx context.Context, userID, text string) (*TodoItem, error) {
	list, err := s.GetOrCreateList(ctx, userID)
	if err != nil {
		return nil, err
	}

	item, err := list.AddItem(text)
	if err != nil {
		return nil, err
	}

	if err := s.store.Save(ctx, list); err != nil {
		return nil, err
	}

	s.publishEvent(ctx, "todo.item.added", userID, item.ItemID, map[string]string{"title": text})
	s.log.Infof("added item %s to list for user %s", item.ItemID, userID)
	return item, nil
}

// ToggleItem toggles an item's completion status.
func (s *Service) ToggleItem(ctx context.Context, userID, itemID string) (*TodoItem, error) {
	list, err := s.store.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	item, err := list.ToggleItem(itemID)
	if err != nil {
		return nil, err
	}

	if err := s.store.Save(ctx, list); err != nil {
		return nil, err
	}

	if item.Completed {
		s.publishEvent(ctx, "todo.item.completed", userID, itemID, nil)
	}
	s.log.Infof("toggled item %s for user %s", itemID, userID)
	return item, nil
}

// RemoveItem removes an item from a user's list.
func (s *Service) RemoveItem(ctx context.Context, userID, itemID string) error {
	list, err := s.store.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if err := list.RemoveItem(itemID); err != nil {
		return err
	}

	if err := s.store.Save(ctx, list); err != nil {
		return err
	}

	s.publishEvent(ctx, "todo.item.removed", userID, itemID, nil)
	s.log.Infof("removed item %s from list for user %s", itemID, userID)
	return nil
}

// UpdateItem updates an item's text.
func (s *Service) UpdateItem(ctx context.Context, userID, itemID, text string) (*TodoItem, error) {
	list, err := s.store.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := list.UpdateItem(itemID, text); err != nil {
		return nil, err
	}

	if err := s.store.Save(ctx, list); err != nil {
		return nil, err
	}

	item, _ := list.GetItem(itemID)
	s.log.Infof("updated item %s for user %s", itemID, userID)
	return item, nil
}

func (s *Service) publishEvent(ctx context.Context, eventType, userID, itemID string, extra map[string]string) {
	if s.publisher == nil {
		return
	}

	payload := map[string]string{
		"event_type": eventType,
		"item_id":    itemID,
	}
	for k, v := range extra {
		payload[k] = v
	}

	env := pubsub.NewEnvelope(audit.Topic, payload).
		WithMetadata("user_id", userID).
		WithMetadata("source", "hatmax")

	if err := s.publisher.Publish(ctx, audit.Topic, env); err != nil {
		s.log.Errorf("cannot publish audit event: %v", err)
	}
}
