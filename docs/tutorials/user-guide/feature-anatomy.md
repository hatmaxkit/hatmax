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

## Follow One Create Across the Boundaries

A create request demonstrates why the layers remain separate:

1. The same-origin middleware admits the state-changing request.
2. The handler parses form values, normalizes text, and reports transport
   errors without calling the service.
3. The service calls `NewInvoice`, which establishes identity, timestamps, and
   durable domain validity.
4. The service asks the `Store` contract to persist the valid invoice.
5. `PostgresStore` maps the model into generated SQLC parameters and executes
   the named query.
6. The handler renders the created row for HTMX or redirects an ordinary
   request to the invoice page.

No layer reaches around the next boundary. The template cannot insert an
invoice, the handler cannot call SQLC, and the store cannot choose an HTTP
status. That constraint keeps one operation understandable from its public
route down to its durable result.

The values passed between boundaries are equally deliberate:

| Boundary | Input | Output |
| --- | --- | --- |
| Browser to handler | form strings, path values, headers | parsed input or safe form errors |
| Handler to service | `InvoiceInput`, identity from context | model value or classified error |
| Service to model | typed creation or update input | valid `Invoice` or validation errors |
| Service to store | valid model or stable identifier | stored result or persistence error |
| Store to SQLC | generated query parameters | generated rows and affected-row counts |
| Handler to template | explicit page or form view | full page or named partial HTML |

Transport-only representations stop at the handler. SQLC representations stop
at the store. The model crosses the service and store boundary without gaining
knowledge of either adapter.

## Preserve Error Meaning

Each layer adds context needed by its caller while preserving error identity:

```text
Postgres or SQLC failure
  -> store wraps the failed persistence operation
  -> service wraps the failed application workflow
  -> handler logs internal context and selects safe HTTP behavior
```

Expected failures take a more specific route. `validation.ValidationErrors`
become `web.FormErrors` and return the form with field feedback.
`ErrNotFound` becomes `404`. An unexpected database or service error becomes a
stable `500` message, while the wrapped detail remains available to logs.

Use `errors.Is` and `errors.As` at the boundary that translates a failure.
Avoid comparing error strings, discarding wrapped causes, or returning
infrastructure messages to the browser.

## Change the Whole Slice

Adding a field is not a model-only edit. A stored, editable `due_date` can
affect:

```text
migration
  -> SQLC select, create, and update queries
  -> generated row and parameter types
  -> model and input
  -> store mappings
  -> domain validation
  -> form parsing and view values
  -> form and row templates
  -> fixtures and boundary tests
```

Optional, computed, or read-only values can remove some obligations, but the
decision must be explicit. Searching only for the Go field name is not enough:
SQL names, HTML input names, labels, and template accessors may use different
representations.

The same principle applies to a new invariant. A client-side `required`,
`minlength`, or similar attribute improves interaction. A durable invariant
also belongs in the model and, when representable, in a Postgres constraint.
Handler tests confirm safe feedback; model and persistence tests confirm that
other callers cannot bypass the rule.

## Test the Contracts, Not the File Count

Tests follow the boundaries that make the feature real:

- model tests cover valid construction, invalid input, and state transitions;
- service tests use a small `Store` fake to prove workflow order and effects;
- handler tests use narrow service and renderer fakes to cover full pages,
  HTMX partials, invalid forms, missing records, and internal failures;
- Postgres store tests use `testhelper.SetupTestDB` to cover queries,
  constraints, mapping, affected-row behavior, and transactions;
- project build and validation gates cover generated SQLC code and explicit
  `main.go` wiring.

A test should fail when its boundary contract regresses. Merely constructing a
type or compiling a mock does not establish feature behavior. Keep fakes local
to the consumer test and implement only the methods that consumer requires.

## Read the Repository Examples as Evidence

The repository examples show working Hatmax behavior, but they were created at
different stages of the project and are not interchangeable blueprints.

The Ticked application provides the closest feature evidence:

- its [list model](../../../examples/ticked/internal/feat/list/model.go),
  [store](../../../examples/ticked/internal/feat/list/store.go), and
  [service](../../../examples/ticked/internal/feat/list/service.go) demonstrate
  a cohesive application feature;
- its SQLC queries, migrations, templates, and explicit composition root show
  the supporting surfaces working together;
- its tests demonstrate domain, service, store, and HTTP behavior.

Ticked predates the consolidated canonical layout. Its HTTP handlers live in
one application-wide package rather than beside each feature. Use those
handlers to inspect real responses, not as authority for new feature
ownership.

The [guide application](../../../examples/guide/main.go) is intentionally
compact so readers can exercise lifecycle and package behavior. Its model,
store, handlers, and wiring share one file; that compression is useful for a
small executable example but is not the structure for a product feature.

The experimental generator applies the same feature anatomy when it changes a
compatible project. Its Hatmax Book makes generation deterministic, but the
Book is generator policy rather than a runtime dependency or a prerequisite
for understanding this chapter. Manually written and generated features must
remain ordinary Go code with the same visible ownership and assembly.

## Continue into Exact Contracts

This chapter defines how the parts fit. Use the reference pages for exact API
behavior:

- [Model](../../reference/model/README.md) for identity and timestamps;
- [Database](../../reference/database/README.md) for connections, migrations,
  and lifecycle;
- [Validation](../../reference/validation/README.md) for field rules and
  structured errors;
- [HTTP](../../reference/http/README.md) and
  [HTMX](../../reference/htmx/README.md) for parsing and response contracts;
- [Application Lifecycle](../../reference/application-lifecycle/README.md)
  for setup and route registration;
- [Test Helper](../../reference/testhelper/README.md) for Postgres integration
  tests;
- [Interfaces and Adapter Ownership](../../explanation/interfaces-and-adapters/README.md)
  for the dependency-direction rationale;
- [Generator](../../reference/generator/README.md) for the experimental
  assisted workflow and its current boundary.

---

[Previous: Presentation Primitives](presentation-primitives.md) ·
[User Guide](README.md) ·
[Next: Persistence and Migrations](persistence-and-migrations.md)
