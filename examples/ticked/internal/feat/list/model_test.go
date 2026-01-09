package list

import (
	"strings"
	"testing"
)

func TestNewTodoList(t *testing.T) {
	list := NewTodoList("user-123")

	if list.ListID == "" {
		t.Error("ListID should not be empty")
	}
	if list.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got %s", list.UserID)
	}
	if len(list.Items) != 0 {
		t.Errorf("expected empty items, got %d", len(list.Items))
	}
	if list.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if list.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestTodoListAddItem(t *testing.T) {
	list := NewTodoList("user-1")

	item, err := list.AddItem("Buy milk")
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	if item.ItemID == "" {
		t.Error("ItemID should not be empty")
	}
	if item.Text != "Buy milk" {
		t.Errorf("expected text 'Buy milk', got %s", item.Text)
	}
	if item.Completed {
		t.Error("new item should not be completed")
	}
	if len(list.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(list.Items))
	}
}

func TestTodoListAddItemEmptyText(t *testing.T) {
	list := NewTodoList("user-1")

	_, err := list.AddItem("")
	if err != ErrTextEmpty {
		t.Errorf("expected ErrTextEmpty, got %v", err)
	}
}

func TestTodoListAddItemTooLong(t *testing.T) {
	list := NewTodoList("user-1")

	longText := strings.Repeat("a", 501)
	_, err := list.AddItem(longText)
	if err != ErrTextTooLong {
		t.Errorf("expected ErrTextTooLong, got %v", err)
	}
}

func TestTodoListAddItemPrependsToList(t *testing.T) {
	list := NewTodoList("user-1")

	item1, _ := list.AddItem("First")
	item2, _ := list.AddItem("Second")

	if len(list.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list.Items))
	}
	// Most recent should be first
	if list.Items[0].ItemID != item2.ItemID {
		t.Error("second item should be at index 0")
	}
	if list.Items[1].ItemID != item1.ItemID {
		t.Error("first item should be at index 1")
	}
}

func TestTodoListUpdateItem(t *testing.T) {
	list := NewTodoList("user-1")
	item, _ := list.AddItem("Original")

	err := list.UpdateItem(item.ItemID, "Updated")
	if err != nil {
		t.Fatalf("UpdateItem failed: %v", err)
	}

	if list.Items[0].Text != "Updated" {
		t.Errorf("expected text 'Updated', got %s", list.Items[0].Text)
	}
}

func TestTodoListUpdateItemNotFound(t *testing.T) {
	list := NewTodoList("user-1")
	list.AddItem("Test")

	err := list.UpdateItem("nonexistent", "Updated")
	if err != ErrItemNotFound {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}

func TestTodoListUpdateItemEmptyText(t *testing.T) {
	list := NewTodoList("user-1")
	item, _ := list.AddItem("Test")

	err := list.UpdateItem(item.ItemID, "")
	if err != ErrTextEmpty {
		t.Errorf("expected ErrTextEmpty, got %v", err)
	}
}

func TestTodoListUpdateItemTooLong(t *testing.T) {
	list := NewTodoList("user-1")
	item, _ := list.AddItem("Test")

	longText := strings.Repeat("a", 501)
	err := list.UpdateItem(item.ItemID, longText)
	if err != ErrTextTooLong {
		t.Errorf("expected ErrTextTooLong, got %v", err)
	}
}

func TestTodoListToggleItem(t *testing.T) {
	list := NewTodoList("user-1")
	item, _ := list.AddItem("Test")

	// Toggle to completed
	toggled, err := list.ToggleItem(item.ItemID)
	if err != nil {
		t.Fatalf("ToggleItem failed: %v", err)
	}
	if !toggled.Completed {
		t.Error("item should be completed after toggle")
	}
	if toggled.CompletedAt == nil {
		t.Error("CompletedAt should be set when completed")
	}

	// Toggle back to uncompleted
	toggled, err = list.ToggleItem(item.ItemID)
	if err != nil {
		t.Fatalf("ToggleItem failed: %v", err)
	}
	if toggled.Completed {
		t.Error("item should be uncompleted after second toggle")
	}
	if toggled.CompletedAt != nil {
		t.Error("CompletedAt should be nil when uncompleted")
	}
}

func TestTodoListToggleItemNotFound(t *testing.T) {
	list := NewTodoList("user-1")
	list.AddItem("Test")

	_, err := list.ToggleItem("nonexistent")
	if err != ErrItemNotFound {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}

func TestTodoListRemoveItem(t *testing.T) {
	list := NewTodoList("user-1")
	item, _ := list.AddItem("Test")

	err := list.RemoveItem(item.ItemID)
	if err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}

	if len(list.Items) != 0 {
		t.Errorf("expected 0 items after remove, got %d", len(list.Items))
	}
}

func TestTodoListRemoveItemNotFound(t *testing.T) {
	list := NewTodoList("user-1")
	list.AddItem("Test")

	err := list.RemoveItem("nonexistent")
	if err != ErrItemNotFound {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}

func TestTodoListGetItem(t *testing.T) {
	list := NewTodoList("user-1")
	item, _ := list.AddItem("Test")

	got, err := list.GetItem(item.ItemID)
	if err != nil {
		t.Fatalf("GetItem failed: %v", err)
	}
	if got.ItemID != item.ItemID {
		t.Errorf("expected ItemID %s, got %s", item.ItemID, got.ItemID)
	}
}

func TestTodoListGetItemNotFound(t *testing.T) {
	list := NewTodoList("user-1")
	list.AddItem("Test")

	_, err := list.GetItem("nonexistent")
	if err != ErrItemNotFound {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}
