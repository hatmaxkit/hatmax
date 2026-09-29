# Feature Anatomy

A Hatmax feature is a vertical slice of application behavior. It owns the
domain values, persistence boundary, workflows, HTTP translation, templates,
and tests required for one cohesive capability.

```text
model -> store -> service -> handler -> templates -> wiring -> tests
```

This is an ownership path, not a sequence of runtime calls. At runtime, a
request enters through the handler, the service coordinates the workflow, the
model protects durable rules, and the store adapts the result to Postgres.
The response returns through a full-page or partial template.

```text
                              +--> model
browser -> handler -> service +--> store -> Postgres
    ^          |
    +----------+-- templates
```

The feature package keeps those decisions close enough to change together.
Hatmax infrastructure remains reusable, and `main.go` makes the assembled
dependencies visible.

## Start from the Complete Shape

For an `invoice` feature, the canonical structure is:

```text
internal/feat/invoice/
├── model.go
├── model_test.go
├── store.go
├── postgres_store.go
├── postgres_store_test.go
├── service.go
├── service_test.go
├── handler.go
└── handler_test.go

assets/migration/postgres/004-invoice.sql
assets/templates/invoice/page.html
assets/templates/invoice/form.html
assets/templates/invoice/row.html
db/queries/invoice.sql
main.go
```

The Go files use one `invoice` package under `internal/feat/invoice`. This is
cohesion, not permission to collapse every responsibility into one type. Each
file keeps a boundary visible, while the package prevents feature behavior
from drifting into generic utility packages or an application-wide handler.

Migrations, SQLC queries, and templates live where their tooling expects them,
but they remain surfaces of the invoice feature. A change is complete only
when all affected surfaces agree.

## Model Durable State

The model owns values and rules that must remain true regardless of whether a
change came from HTTP, a scheduled task, or a test:

```go
type Invoice struct {
	ID        string
	Number    string
	Notes     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type InvoiceInput struct {
	Number string
	Notes  string
}

func NewInvoice(input InvoiceInput) (*Invoice, error) {
	now := model.Now()
	invoice := &Invoice{
		ID:        model.NewID(),
		Number:    input.Number,
		Notes:     input.Notes,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := invoice.Validate(); err != nil {
		return nil, err
	}

	return invoice, nil
}
```

Construction establishes identity, timestamps, and valid initial state.
Update methods apply mutable input, validate it, and set the update time only
after validation succeeds. `Validate` returns Hatmax
`validation.ValidationErrors` for user-correctable field failures.

The model does not import HTTP, templates, SQLC row types, or database
handles. Persistence constraints repeat applicable durable invariants because
concurrent writes can bypass a service-side pre-check.

## Own the Persistence Contract

The feature defines the behavior it needs from storage:

```go
var ErrNotFound = errors.New("invoice not found")

type Store interface {
	List(context.Context) ([]Invoice, error)
	Get(context.Context, string) (*Invoice, error)
	Create(context.Context, *Invoice) error
	Update(context.Context, *Invoice) error
	Delete(context.Context, string) error
}
```

This consumer-owned interface is small enough for service tests and stable
enough for a concrete adapter. `PostgresStore` implements it through generated
SQLC queries. The adapter owns row-to-model mapping, contextual error wrapping,
and translation of missing rows or zero affected rows to `ErrNotFound`.

The store constructor retains the Hatmax database provider without performing
I/O. Its `Start` method acquires the live connection and constructs the SQLC
query set after the database has started. `Stop` keeps lifecycle slices
aligned even when the store itself owns no connection to close.

Postgres persistence therefore has three explicit representations:

- a reversible migration defines schema, constraints, and indexes;
- `db/queries/invoice.sql` defines named SQLC operations;
- `PostgresStore` maps generated rows and parameters to the feature model.

The handler never calls SQLC or the database directly. The store does not
decide who may create an invoice or which HTML should be returned.

## Coordinate Work in the Service

The service owns the application workflow:

```go
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (service *Service) Create(
	ctx context.Context,
	input InvoiceInput,
) (*Invoice, error) {
	invoice, err := NewInvoice(input)
	if err != nil {
		return nil, err
	}

	if err := service.store.Create(ctx, invoice); err != nil {
		return nil, fmt.Errorf("create invoice: %w", err)
	}

	return invoice, nil
}
```

The workflow validates the domain value before persistence and wraps
infrastructure failures without destroying their classification. Update loads
the current entity, applies its domain method, and persists the valid result.
Delete delegates to the store while preserving `ErrNotFound` through wrapping.

Events, mail, jobs, and other effects become explicit service dependencies
only when the feature requires them. They are not hidden side effects of a
generic CRUD base type.

## Translate HTTP in the Handler

The feature-owned handler declares the narrow behavior it consumes:

```go
type FeatureService interface {
	List(context.Context) ([]Invoice, error)
	Create(context.Context, InvoiceInput) (*Invoice, error)
	Update(context.Context, string, InvoiceInput) (*Invoice, error)
	Delete(context.Context, string) error
}

type TemplateRenderer interface {
	Render(http.ResponseWriter, string, string, interface{})
	RenderPartial(http.ResponseWriter, string, string, interface{})
}

type Handler struct {
	service   FeatureService
	templates TemplateRenderer
	log       log.Logger
}
```

These interfaces belong to the consumer that needs them. The production
`Service` and `web.TemplateManager` satisfy them, while handler tests provide
small fakes without replacing unrelated infrastructure.

`RegisterRoutes` owns the feature paths:

```go
func (handler *Handler) RegisterRoutes(router chi.Router) {
	router.Get("/invoices", handler.page)
	router.Post("/invoices", handler.create)
	router.Put("/invoices/{id}", handler.update)
	router.Delete("/invoices/{id}", handler.delete)
}
```

Each handler parses and normalizes transport values, calls one service
workflow, translates classified failures, and selects a page, partial,
redirect, or status response. It logs internal failure context but returns a
stable, safe browser message.

Invalid form input renders `form.html` with `web.FormErrors`. A successful
HTMX create or update renders `row.html`; an ordinary request redirects to the
feature page. A successful HTMX delete uses `htmx.RespondDelete`, while an
ordinary delete redirects. The application-wide same-origin middleware
protects all state-changing routes before they reach the handler.

## Keep Templates with the Feature Vocabulary

The page, form, and row templates are three views of the same feature:

- `page.html` composes the form and current rows;
- `form.html` preserves submitted values and field feedback;
- `row.html` is both the list representation and the stable HTMX replacement
  unit for one invoice.

Templates use Hatmax UI and HTMX helpers when a matching helper exists. They
do not own domain rules, query the store, or keep a second client-side copy of
application state. Stable element IDs derive from the feature and entity ID,
so a handler response has an unambiguous target.

## Wire the Slice in `main`

The composition root constructs concrete values in dependency order:

```go
invoiceStore := invoicefeat.NewPostgresStore(database)
invoiceService := invoicefeat.NewService(invoiceStore)
invoiceHandler := invoicefeat.NewHandler(
	invoiceService,
	tmplMgr,
	logger,
)

deps := []any{
	database,
	migrator,
	tmplMgr,
	invoiceStore,
	invoiceHandler,
}
```

The plain service is constructed explicitly but does not appear in `deps`
because it has no lifecycle or route-registration behavior. The store follows
the database, and the handler follows every dependency it consumes.
`app.Setup` extracts lifecycle and registrar functions from that visible
ordered list; it does not infer a hidden graph.

`main.go` remains one `main` function and the composition root. Feature
behavior stays in the feature package, and helper builders do not accumulate
beside the entrypoint.

---

[Previous: Presentation Primitives](presentation-primitives.md) ·
[User Guide](README.md)
