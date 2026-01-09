-- name: UpsertTodoList :exec
INSERT INTO todo_lists (id, user_id, created_at, updated_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    updated_at = EXCLUDED.updated_at;

-- name: GetTodoListByUserID :one
SELECT id, user_id, created_at, updated_at
FROM todo_lists
WHERE user_id = $1;

-- name: DeleteTodoList :exec
DELETE FROM todo_lists WHERE id = $1;

-- name: InsertTodoItem :exec
INSERT INTO todo_items (id, list_id, text, completed, created_at, completed_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateTodoItem :exec
UPDATE todo_items
SET text = $2, completed = $3, completed_at = $4
WHERE id = $1;

-- name: DeleteTodoItem :exec
DELETE FROM todo_items WHERE id = $1;

-- name: DeleteTodoItemsByListID :exec
DELETE FROM todo_items WHERE list_id = $1;

-- name: GetTodoItemsByListID :many
SELECT id, list_id, text, completed, created_at, completed_at
FROM todo_items
WHERE list_id = $1
ORDER BY created_at DESC;

-- name: GetTodoItemByID :one
SELECT id, list_id, text, completed, created_at, completed_at
FROM todo_items
WHERE id = $1;
