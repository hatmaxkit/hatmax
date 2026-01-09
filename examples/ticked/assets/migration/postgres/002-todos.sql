-- +migrate Up
CREATE TABLE IF NOT EXISTS todo_lists (
    id TEXT PRIMARY KEY,
    user_id TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS todo_items (
    id TEXT PRIMARY KEY,
    list_id TEXT NOT NULL,
    text TEXT NOT NULL CHECK (length(text) <= 500),
    completed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_todo_items_list_id ON todo_items(list_id);

-- +migrate Down
DROP TABLE IF EXISTS todo_items;
DROP TABLE IF EXISTS todo_lists;
