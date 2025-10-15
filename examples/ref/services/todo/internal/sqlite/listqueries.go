package sqlite

const (
	// Queries for List aggregate root operations

	// QueryCreateListRoot creates a new List aggregate root record.
	QueryCreateListRoot = `INSERT INTO lists (id, description, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`

	// QueryGetListRoot retrieves a List aggregate root record by ID.
	QueryGetListRoot = `SELECT id, description, name, created_at, updated_at FROM lists WHERE id = ?`

	// QueryUpdateListRoot updates an existing List aggregate root record.
	QueryUpdateListRoot = `UPDATE lists SET description = ?, name = ?, updated_at = ? WHERE id = ?`

	// QueryDeleteListRoot deletes a List aggregate root record by ID.
	QueryDeleteListRoot = `DELETE FROM lists WHERE id = ?`

	// QueryListListRoot lists all List aggregate root records.
	QueryListListRoot = `SELECT id FROM lists ORDER BY created_at DESC`

	// Queries for List's Items child entities

	// QueryCreateListItems creates new Item records for a List aggregate.
	QueryCreateListItems = `INSERT INTO items (text, done) VALUES `

	// QueryGetListItems retrieves all Item records for a specific List aggregate.
	QueryGetListItems = `SELECT text, done FROM items WHERE List_id = ? ORDER BY created_at`

	// QueryUpdateListItem updates an existing Item record within a List aggregate.
	QueryUpdateListItem = `UPDATE items SET text = ?, done = ? WHERE id = ?`

	// QueryDeleteListItems deletes all Item records for a specific List aggregate.
	QueryDeleteListItems = `DELETE FROM items WHERE List_id = ?`

	// QueryDeleteListItemsByIDs deletes specific Item records by their IDs.
	QueryDeleteListItemsByIDs = `DELETE FROM items WHERE id IN `

	// Helper query parts for batch operations
	ItemValuePlaceholder = `(?, ?)`

	// Queries for List's Tags child entities

	// QueryCreateListTags creates new Tag records for a List aggregate.
	QueryCreateListTags = `INSERT INTO tags (name, color) VALUES `

	// QueryGetListTags retrieves all Tag records for a specific List aggregate.
	QueryGetListTags = `SELECT name, color FROM tags WHERE List_id = ? ORDER BY created_at`

	// QueryUpdateListTag updates an existing Tag record within a List aggregate.
	QueryUpdateListTag = `UPDATE tags SET name = ?, color = ? WHERE id = ?`

	// QueryDeleteListTags deletes all Tag records for a specific List aggregate.
	QueryDeleteListTags = `DELETE FROM tags WHERE List_id = ?`

	// QueryDeleteListTagsByIDs deletes specific Tag records by their IDs.
	QueryDeleteListTagsByIDs = `DELETE FROM tags WHERE id IN `

	// Helper query parts for batch operations
	TagValuePlaceholder = `(?, ?)`
)
