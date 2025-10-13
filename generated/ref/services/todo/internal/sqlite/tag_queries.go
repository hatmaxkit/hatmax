package sqlite

const (
	// QueryCreateTag creates a new Tag record.
	QueryCreateTag = `INSERT INTO tags (id, name, color, created_at, updated_at, created_by, updated_by) VALUES (?, ?, ?, ?, ?, ?, ?)`

	// QueryGetTag retrieves a Tag record by ID.
	QueryGetTag = `SELECT id, name, color, created_at, updated_at, created_by, updated_by FROM tags WHERE id = ?`

	// QueryUpdateTag updates an existing Tag record.
	// This is a full update; consider optimizing for partial updates if needed.
	QueryUpdateTag = `UPDATE tags SET name = ?, color = ?, updated_at = ?, updated_by = ? WHERE id = ?`

	// QueryDeleteTag deletes a Tag record by ID.
	QueryDeleteTag = `DELETE FROM tags WHERE id = ?`

	// QueryListTag lists all Tag records.
	QueryListTag = `SELECT id, name, color, created_at, updated_at, created_by, updated_by FROM tags`
)
