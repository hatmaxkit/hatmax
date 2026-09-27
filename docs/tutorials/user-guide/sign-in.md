# Sign In

Create an account, sign in, and open a page that only that signed-in request
can see.

This chapter keeps Postgres, the template manager, and same-origin protection.
It does not save the name from Accept a Form.

The account flow is in
[Authentication](../../reference/authentication/index.md).
Password hashes and session tokens in that flow come from `model`.
[Crypto](../../reference/crypto/index.md) is the separate set of primitives
for authenticated strings, lookup hashes, PASETO, and TOTP. This chapter does
not call those functions.

## Store users and sessions

`auth.NewService` needs an `auth.Queries` implementation. Add `authstore.go`.
`Start` runs after the database connection is open and creates two tables.
`GetUserByEmail` and `GetSessionByToken` return `sql.ErrNoRows` when the row
is missing. `Signup` treats any other error as a failure to check the email.

```go
package main

import (
	"context"
	"database/sql"
	"time"

	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/db"
)

type queries struct {
	database *db.Database
}

func (q *queries) Start(ctx context.Context) error {
	_, err := q.database.GetDB().ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS guide_users (
			id text PRIMARY KEY,
			email text UNIQUE NOT NULL,
			password_hash text NOT NULL,
			active boolean NOT NULL,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL
		);
		CREATE TABLE IF NOT EXISTS guide_sessions (
			id text PRIMARY KEY,
			user_id text NOT NULL,
			token text UNIQUE NOT NULL,
			expires_at timestamptz NOT NULL,
			created_at timestamptz NOT NULL
		);
	`)

	return err
}

func (q *queries) Stop(context.Context) error { return nil }

func (q *queries) CreateUser(ctx context.Context, id, email, passwordHash string, createdAt, updatedAt time.Time) (*auth.User, error) {
	_, err := q.database.GetDB().ExecContext(ctx, `
		INSERT INTO guide_users (id, email, password_hash, active, created_at, updated_at)
		VALUES ($1, $2, $3, true, $4, $5)
	`, id, email, passwordHash, createdAt, updatedAt)
	if err != nil {
		return nil, err
	}

	return &auth.User{
		ID: id, Email: email, PasswordHash: passwordHash, Active: true,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}

func (q *queries) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	return q.scanUser(ctx, `SELECT id, email, password_hash, active FROM guide_users WHERE email = $1`, email)
}

func (q *queries) GetUserByID(ctx context.Context, id string) (*auth.User, error) {
	return q.scanUser(ctx, `SELECT id, email, password_hash, active FROM guide_users WHERE id = $1`, id)
}

func (q *queries) scanUser(ctx context.Context, query, arg string) (*auth.User, error) {
	var user auth.User
	err := q.database.GetDB().QueryRowContext(ctx, query, arg).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Active,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (q *queries) CreateSession(ctx context.Context, id, userID, token string, expiresAt, createdAt time.Time) (*auth.Session, error) {
	_, err := q.database.GetDB().ExecContext(ctx, `
		INSERT INTO guide_sessions (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, id, userID, token, expiresAt, createdAt)
	if err != nil {
		return nil, err
	}

	return &auth.Session{ID: id, UserID: userID, Token: token, ExpiresAt: expiresAt, CreatedAt: createdAt}, nil
}

func (q *queries) GetSessionByToken(ctx context.Context, token string) (*auth.Session, error) {
	var session auth.Session
	err := q.database.GetDB().QueryRowContext(ctx, `
		SELECT id, user_id, token, expires_at, created_at
		FROM guide_sessions WHERE token = $1
	`, token).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (q *queries) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := q.database.GetDB().ExecContext(ctx, `DELETE FROM guide_sessions WHERE id = $1`, sessionID)

	return err
}

func (q *queries) DeleteExpiredSessions(ctx context.Context) error {
	_, err := q.database.GetDB().ExecContext(ctx, `DELETE FROM guide_sessions WHERE expires_at < now()`)

	return err
}
```

Pass `queries` to `app.Setup` after `database` and before the template manager.

## Add the pages

Add `assets/templates/home/signin.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Sign in</title></head>
<body>
  <p id="result">{{.Message}}</p>
  <form method="post" action="/signup">
    <input name="email" type="email">
    <input name="password" type="password">
    <button type="submit">Create account</button>
  </form>
  <form method="post" action="/signin">
    <input name="email" type="email">
    <input name="password" type="password">
    <button type="submit">Sign in</button>
  </form>
</body>
</html>
```

Add `assets/templates/home/private.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Private</title></head>
<body><p id="who">{{.Email}}</p></body>
</html>
```

Give `pages` an `account *auth.Service` field and set it to
`auth.NewService(store, cfg, logger)` before `app.Setup`. The default
password minimum is 8 characters. `password1` meets it. `Signup` stores a
bcrypt hash from `model.HashPassword`. `Signin` compares that hash and stores
a session whose token is a new model ID.

```go
func (p *pages) RegisterRoutes(r chi.Router) {
	r.Get("/signin", p.signinPage)
	r.Post("/signup", p.signup)
	r.Post("/signin", p.signin)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(p.account))
		r.Get("/private", p.private)
	})
}

func (p *pages) signinPage(w http.ResponseWriter, r *http.Request) {
	p.templates.Render(w, "home", "signin", map[string]string{})
}

func (p *pages) signup(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	message := "Account created"
	if err == nil {
		_, err = p.account.Signup(r.Context(), r.FormValue("email"), r.FormValue("password"))
	}
	if err != nil {
		message = err.Error()
	}
	p.templates.Render(w, "home", "signin", map[string]string{"Message": message})
}

func (p *pages) signin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	session, err := p.account.Signin(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		p.templates.Render(w, "home", "signin", map[string]string{"Message": err.Error()})
		return
	}
	maxAge := int(time.Until(session.ExpiresAt).Seconds())
	auth.SetSessionCookie(w, session.Token, maxAge)
	http.Redirect(w, r, "/private", http.StatusSeeOther)
}

func (p *pages) private(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r.Context())
	p.templates.Render(w, "home", "private", map[string]string{"Email": user.Email})
}
```

`RequireAuth` redirects to `/signin` with status `303` when the `session`
cookie is missing or `ValidateSession` fails. `SetSessionCookie` marks the
cookie `Secure`. A browser on `http://localhost` does not send that cookie
back. The check below uses curl, which sends the cookie it stored.

Keep `middleware.RequireSameOrigin` on the router. The POST requests need
`Origin: http://localhost:8080`.

## Check the result

Run the process against the Postgres from Add Postgres.

Without a cookie, `/private` redirects:

```sh
curl -sS -D - -o /dev/null http://localhost:8080/private
```

The status is `303` and `Location` is `/signin`.

Create the account, sign in, and open the private page with the saved cookie:

```sh
curl -c /tmp/guide.cookies -b /tmp/guide.cookies \
  -H 'Origin: http://localhost:8080' \
  -d 'email=ada@example.com&password=password1' \
  http://localhost:8080/signup

curl -c /tmp/guide.cookies -b /tmp/guide.cookies \
  -H 'Origin: http://localhost:8080' \
  -d 'email=ada@example.com&password=password1' \
  -D - -o /dev/null \
  http://localhost:8080/signin

curl -sS -b /tmp/guide.cookies http://localhost:8080/private
```

The sign-in response is `303` to `/private` and includes `Set-Cookie`.
The last command prints the private page with `ada@example.com`. The process
is still running. Stop it with Ctrl+C.
