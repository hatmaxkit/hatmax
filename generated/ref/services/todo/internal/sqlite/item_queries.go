package sqlite

const (
	// QueryCreateItem creates a new Item record.
	QueryCreateItem = `INSERT INTO items (id, text, done, created_at, updated_at, created_by, updated_by) VALUES (?, ?, ?, ?, ?, ?, ?)`

	// QueryGetItem retrieves a Item record by ID.
	QueryGetItem = `SELECT id, text, done, created_at, updated_at, created_by, updated_by FROM items WHERE id = ?`

	// QueryUpdateItem updates an existing Item record.
	// This is a full update; consider optimizing for partial updates if needed.
	QueryUpdateItem = `UPDATE items SET text = ?, done = ?, updated_at = ?, updated_by = ? WHERE id = ?`

	// QueryDeleteItem deletes a Item record by ID.
	QueryDeleteItem = `DELETE FROM items WHERE id = ?`

	// QueryListItem lists all Item records.
	QueryListItem = `SELECT id, text, done, created_at, updated_at, created_by, updated_by FROM items`
)
