# Save a Record

Store one note that is still listed after the process starts again.

This chapter keeps Postgres from Add Postgres. The note page is not behind
sign-in, so the restart check does not depend on the session cookie.

The contracts are in
[Model](../../reference/model/index.md),
[Slug](../../reference/slug/index.md),
[Seed](../../reference/seed/index.md), and
[Database](../../reference/database/index.md).

## Create the table

Add a component that creates `guide_notes` in `Start`, after the database
connection is open. Put it in `app.Setup` immediately after `database`.

```go
type notes struct {
	database *db.Database
}

func (n *notes) Start(ctx context.Context) error {
	_, err := n.database.GetDB().ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS guide_notes (
			id text PRIMARY KEY,
			title text NOT NULL,
			slug text NOT NULL,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL
		)
	`)

	return err
}

func (n *notes) Stop(context.Context) error { return nil }
```

## Seed one note

A seeder runs once. Later starts skip it because its name is in `_seeds`.
`seed.NewRunner` needs the database provider, this seeder, and the logger.
Pass the runner to `app.Setup` after `notes`, so the table exists before the
insert.

```go
type welcomeNote struct {
	database *db.Database
}

func (w *welcomeNote) Name() string { return "guide-welcome-note" }

func (w *welcomeNote) Seed(ctx context.Context) error {
	id := model.NewID()
	parsed, err := model.ParseID(id)
	if err != nil {
		return err
	}

	now := model.Now()
	_, err = w.database.GetDB().ExecContext(ctx, `
		INSERT INTO guide_notes (id, title, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, id, "Welcome", slug.Generate("Welcome", parsed), now, now)

	return err
}
```

`slug.Generate` appends the first 8 characters of the UUID. `Welcome` becomes
`welcome-` plus those characters.

## Save from the request

Add `assets/templates/home/notes.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Notes</title></head>
<body>
  <form method="post" action="/notes">
    <input name="title">
    <button type="submit">Save</button>
  </form>
  <ul id="notes">
    {{range .Notes}}<li>{{.Title}} <code>{{.Slug}}</code></li>{{end}}
  </ul>
</body>
</html>
```

`GET /notes` lists every row. `POST /notes` inserts the submitted title with
a new id, `model.Now` for both timestamps, and `slug.Generate` for the slug.
Keep `middleware.RequireSameOrigin` on the router, so the POST needs
`Origin: http://localhost:8080`.

```go
func (n *notes) Insert(ctx context.Context, title string) error {
	id := model.NewID()
	parsed, err := model.ParseID(id)
	if err != nil {
		return err
	}

	now := model.Now()
	_, err = n.database.GetDB().ExecContext(ctx, `
		INSERT INTO guide_notes (id, title, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, id, title, slug.Generate(title, parsed), now, now)

	return err
}
```

## Check the result

Run the process against the Postgres from Add Postgres.

```sh
curl -sS http://localhost:8080/notes
```

The list contains `Welcome`.

Save a note, stop the process, start it again, and list the notes:

```sh
curl -sS -H 'Origin: http://localhost:8080' \
  -d 'title=Hello' http://localhost:8080/notes
```

Stop the process with Ctrl+C and start it again.

```sh
curl -sS http://localhost:8080/notes
```

The list still contains `Welcome` and `Hello`. `Welcome` was not inserted a
second time. Each title is followed by its slug.
