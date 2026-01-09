package list

import (
	"errors"
	"time"

	"github.com/hatmaxkit/hatmax/model"
)

var (
	ErrItemNotFound = errors.New("item not found")
	ErrTextTooLong  = errors.New("text exceeds 500 characters")
	ErrTextEmpty    = errors.New("text cannot be empty")
)

// TodoList is the aggregate root for a user's todo list.
type TodoList struct {
	ListID    string
	UserID    string
	Items     []TodoItem
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TodoItem represents a single todo item within a list.
type TodoItem struct {
	ItemID      string
	Text        string
	Completed   bool
	CreatedAt   time.Time
	CompletedAt *time.Time
}

// NewTodoList creates a new empty todo list for a user.
func NewTodoList(userID string) *TodoList {
	now := model.Now()
	return &TodoList{
		ListID:    model.NewID(),
		UserID:    userID,
		Items:     []TodoItem{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddItem adds a new item to the list.
func (l *TodoList) AddItem(text string) (*TodoItem, error) {
	if text == "" {
		return nil, ErrTextEmpty
	}
	if len(text) > 500 {
		return nil, ErrTextTooLong
	}

	item := TodoItem{
		ItemID:    model.NewID(),
		Text:      text,
		Completed: false,
		CreatedAt: model.Now(),
	}

	l.Items = append([]TodoItem{item}, l.Items...)
	l.UpdatedAt = model.Now()

	return &item, nil
}

// UpdateItem updates an existing item's text.
func (l *TodoList) UpdateItem(itemID, text string) error {
	if text == "" {
		return ErrTextEmpty
	}
	if len(text) > 500 {
		return ErrTextTooLong
	}

	for i := range l.Items {
		if l.Items[i].ItemID == itemID {
			l.Items[i].Text = text
			l.UpdatedAt = model.Now()
			return nil
		}
	}

	return ErrItemNotFound
}

// ToggleItem toggles the completion status of an item.
func (l *TodoList) ToggleItem(itemID string) (*TodoItem, error) {
	for i := range l.Items {
		if l.Items[i].ItemID == itemID {
			l.Items[i].Completed = !l.Items[i].Completed
			if l.Items[i].Completed {
				now := model.Now()
				l.Items[i].CompletedAt = &now
			} else {
				l.Items[i].CompletedAt = nil
			}
			l.UpdatedAt = model.Now()
			return &l.Items[i], nil
		}
	}

	return nil, ErrItemNotFound
}

// RemoveItem removes an item from the list.
func (l *TodoList) RemoveItem(itemID string) error {
	for i := range l.Items {
		if l.Items[i].ItemID == itemID {
			l.Items = append(l.Items[:i], l.Items[i+1:]...)
			l.UpdatedAt = model.Now()
			return nil
		}
	}

	return ErrItemNotFound
}

// GetItem returns an item by ID.
func (l *TodoList) GetItem(itemID string) (*TodoItem, error) {
	for i := range l.Items {
		if l.Items[i].ItemID == itemID {
			return &l.Items[i], nil
		}
	}
	return nil, ErrItemNotFound
}
