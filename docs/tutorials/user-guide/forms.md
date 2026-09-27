# Accept a Form

Accept one name and reject a name that is missing or shorter than two
characters. A POST from another site is refused before validation runs.

This chapter keeps the page and the Postgres component from Serve a Page.
It does not save the name.

The contracts are in
[Validation](../../reference/validation/index.md),
[UI](../../reference/ui/index.md), and
[Middleware](../../reference/middleware/index.md).

## Protect the POST

Pass `middleware.RequireSameOrigin` to `app.NewRouter`. `GET` and `HEAD` are
allowed through. Any other method needs an `Origin` or `Referer` for this
host. A cross-site POST receives `403`.

```go
router := app.NewRouter(
	logger,
	app.WithMiddleware(middleware.RequireSameOrigin),
	app.WithPing(),
)
```

## Replace the page

Replace `assets/templates/home/page.html` with:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Name</title>
</head>
<body>
  {{.FormOpen}}
  <label>Name <input name="name" value="{{.Name}}"></label>
  <button type="submit">Save</button>
  {{.FormClose}}
  <p id="result">{{.Message}}</p>
</body>
</html>
```

`ui.NewForm` renders the `<form>` tags. The default method is `post` and the
action is `/name`. No CSRF token is set, so the form has no hidden token.

## Validate the name

Add the POST handler on the page component. `validation.Field` collects the
errors. An empty value fails `Required`. A one-character value fails
`MinLength`. Two or more characters are accepted.

```go
import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/middleware"
	"hatmax.adrianpk.com/ui"
	"hatmax.adrianpk.com/validation"
)

type pageView struct {
	FormOpen  template.HTML
	FormClose template.HTML
	Name      string
	Message   string
}

func (p *pages) RegisterRoutes(r chi.Router) {
	r.Get("/", p.home)
	r.Post("/name", p.saveName)
}

func (p *pages) formTags() (template.HTML, template.HTML) {
	form := ui.NewForm().Post().Action("/name")

	return form.Open(), form.Close()
}

func (p *pages) home(w http.ResponseWriter, r *http.Request) {
	open, close := p.formTags()
	p.templates.Render(w, "home", "page", pageView{
		FormOpen:  open,
		FormClose: close,
	})
}

func (p *pages) saveName(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	name := r.FormValue("name")
	view := pageView{Name: name}
	view.FormOpen, view.FormClose = p.formTags()

	errs := validation.Field("name", name).Required().MinLength(2).Errors()
	if errs.HasErrors() {
		view.Message = errs.Error()
	} else {
		view.Message = "Accepted " + name
	}

	p.templates.Render(w, "home", "page", view)
}
```

Keep `database`, `templates`, and `site` in `app.Setup`, in that order.

## Check the result

Run the process against the Postgres from Add Postgres.

A POST with no `Origin` is refused:

```sh
curl -sS -o /dev/null -w '%{http_code}\n' -X POST localhost:8080/name -d 'name=Ada'
```

The status is `403`.

A same-origin POST with an empty name is rejected, and `Ada` is accepted:

```sh
curl -sS -X POST -H 'Origin: http://localhost:8080' -d 'name=' localhost:8080/name
curl -sS -X POST -H 'Origin: http://localhost:8080' -d 'name=A' localhost:8080/name
curl -sS -X POST -H 'Origin: http://localhost:8080' -d 'name=Ada' localhost:8080/name
```

The first response contains `name: is required`. The second contains
`name: must be at least 2 characters`. The third contains `Accepted Ada`.

Open `http://localhost:8080/`, submit the empty form, then submit `Ada`. The
same messages appear in `#result`. The process is still running. Stop it
with Ctrl+C.
