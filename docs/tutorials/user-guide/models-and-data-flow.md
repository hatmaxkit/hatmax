<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Models and Data Flow

A Hatmax application carries one fact through several representations. Each
representation belongs to a boundary and should contain only what that
boundary needs.

```text
form values -> feature input -> domain model -> SQLC parameters -> Postgres
Postgres    -> SQLC row      -> domain model -> view model    -> HTML
```

The names may align, but the types are not interchangeable. Explicit
conversion prevents HTTP, generated persistence, and presentation details
from becoming the domain model.

## Distinguish the Representations

For an invoice, the application can use:

| Representation | Owner | Purpose |
| --- | --- | --- |
| `web.FormValues` | HTTP handler | Read strings and transport types from a request. |
| `FormView` | HTTP handler | Preserve submitted text and safe validation feedback. |
| `InvoiceInput` | feature model | Carry typed values for create or update. |
| `Invoice` | feature model | Represent valid durable application state. |
| `dal.CreateInvoiceParams` | generated DAL | Match one SQL query's arguments. |
| `dal.Invoice` | generated DAL | Match selected database columns and nullability. |
| `InvoiceView` | presentation code | Carry display-ready application data to templates. |

Do not reuse the SQLC row as the domain model merely because their fields are
currently similar. A query can change its selected columns, join auxiliary
data, or expose `sql.Null*` types without changing the meaning of an invoice.
Likewise, a view can add labels and formatted strings without polluting
durable state.

## Establish Identity and Time in the Model

Hatmax supplies consistent primitives for identifiers and timestamps:

```go
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

`model.NewID` returns a UUID string and `model.ParseID` validates one at an
input boundary. `model.Now` uses UTC. `SetCreated` and `SetUpdated` are useful
when a model exposes those timestamps as separate fields, but constructors and
domain methods still decide when a state transition warrants a change.

An update should not replace identity or creation time:

```go
func (invoice *Invoice) Update(input InvoiceInput) error {
	invoice.Number = input.Number
	invoice.Notes = input.Notes

	if err := invoice.Validate(); err != nil {
		return err
	}

	invoice.UpdatedAt = model.Now()

	return nil
}
```

For non-trivial transitions, validate a candidate or restore previous values
before returning an error. A failed method must not leave a long-lived model
in an invalid partially updated state.

## Normalize Before Domain Validation

The HTTP handler converts raw values into `InvoiceInput`. Normalization removes
transport noise; validation decides whether the normalized value is allowed:

```go
input := InvoiceInput{
	Number: validation.NormalizeText(form.String("number")),
	Notes:  validation.NormalizeText(form.String("notes")),
}
```

The model validates durable rules so every caller receives the same result.
The handler may detect type parsing failures earlier and add browser-specific
constraints for faster feedback. Postgres repeats constraints needed to
protect concurrent durable writes.

Optionality must remain consistent. An optional empty string can remain an
empty domain value and a `NOT NULL DEFAULT ''` column, or it can become a
pointer and a nullable column. Choose one representation deliberately and map
it in the store. Hatmax `model.NullUUID` and `model.FromNullUUID` help convert
optional UUID strings at that adapter boundary.

## Map at the Postgres Adapter

The store is the only feature layer that knows both domain and DAL types:

```go
func invoiceFromRow(row dal.Invoice) Invoice {
	return Invoice{
		ID:        row.ID,
		Number:    row.Number,
		Notes:     row.Notes,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
```

Create and update methods perform the inverse mapping into their specific SQLC
parameter types. Keep both directions explicit. Reflection or generic field
copying hides schema differences and makes a newly added column appear to work
while silently losing data.

The store translates persistence-specific absence into the feature's
`ErrNotFound`. It does not convert validation failures, select templates, or
construct browser messages.

## Let the Service Define the Workflow

On create, the service constructs a valid entity and persists it. On update,
it loads current state, applies a domain transition, and persists the result:

```text
Create: input -> constructor -> valid model -> store.Create
Update: id -> store.Get -> model.Update -> store.Update
Delete: id -> authorization and workflow rules -> store.Delete
```

Loading before update is intentional. The domain method sees current state,
preserves immutable values, and can enforce transition rules. An SQL-only
partial update can be appropriate for a specialized operation, but it remains
a named feature workflow rather than a shortcut around domain ownership.

The request context travels through handler, service, store, SQLC, and
Postgres so cancellation and deadlines reach the actual I/O. Do not replace it
with `context.Background` inside the feature path.

## Model Aggregates with Transactional Persistence

Some features persist more than one table as one domain unit. Ticked models a
todo list and its items as an aggregate. Its store loads the list and items
into one model and saves changes inside a transaction.

The aggregate decides which item transitions are valid. The store decides how
to synchronize the rows atomically. A handler still calls a service workflow;
it does not issue one request per table.

Use aggregate boundaries only when the data changes under one consistency
rule. Unrelated records do not become one aggregate merely because a page
displays them together.

## Treat Derived Values Deliberately

Some stored values derive from domain data. The `slug` package creates an
ASCII, length-bounded slug and can append an ID prefix for uniqueness:

```go
parsedID, err := model.ParseID(invoice.ID)
if err != nil {
	return err
}

invoice.Slug = slug.Generate(invoice.Number, parsedID)
```

Decide whether the slug is durable identity, a mutable presentation alias, or
a computed value. That decision determines whether it belongs in the model,
schema, unique constraints, update workflow, and redirects. Do not regenerate
a durable public URL accidentally when display text changes.

Seed references solve a different problem. `seed.RefMap` associates symbolic
names with IDs created during an ordered seed run so later seeders can connect
records without hard-coded UUIDs. Those references are bootstrap coordination,
not application-facing identity.

## Trace a Field Before Calling the Change Complete

For every stored field, trace both directions:

```text
HTML name -> form value -> normalized input -> model field
  -> SQLC parameter -> column

column -> SQLC row -> model field -> view value -> escaped HTML
```

Then check the cross-cutting surfaces: migration constraints, query columns,
store mapping, domain validation, service workflows, form errors, templates,
fixtures, and tests. This trace is more reliable than editing one struct and
following compiler failures, because several representations remain strings
and can compile while disagreeing semantically.

For exact primitives, see [Model](../../reference/model/README.md),
[Validation](../../reference/validation/README.md),
[Slug](../../reference/slug/README.md), and
[Seed](../../reference/seed/README.md). The preceding
[Feature Anatomy](feature-anatomy.md) places these representations in the
complete vertical slice.

---

[Previous: Persistence and Migrations](persistence-and-migrations.md) ·
[User Guide](README.md) ·
[Next: Identity and Sessions](identity-and-sessions.md)
