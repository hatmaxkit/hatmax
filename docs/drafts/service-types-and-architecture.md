# Service Types and Architecture

**Status:** Draft | **Updated:** 2025-10-06 | **Version:** 0.1

## Overview

Hatmax supports four service types that cover the full spectrum of backend architectures: `atom`, `domain`, `composite`, and `web`. Each type serves a specific architectural purpose and generates different code patterns, enabling clean separation of concerns and natural evolution paths as complexity grows.

- **atom**: Single-entity CRUD with aggregates support
- **domain**: Multi-entity bounded context with complex business logic  
- **composite**: API orchestrator without database
- **web**: Presentation layer with templates

## Current Implementation

### `atom` Service ✅

Currently implemented and fully functional. Generates CRUD operations for a single entity with aggregate support, multiple repository implementations, and declarative handlers.

**YAML Structure:**
```yaml
version: 0.1
name: "ref"
package: "github.com/adrianpk/hatmax-ref"
services:
  todo:
    kind: atom
    repo_impl: [sqlite, mongo]
    auth:
      enabled: true
      mode: development
      required_scopes: ["read:todos", "write:todos"]
    models:
      Item:
        options:
          audit: true
          lifecycle: [before_create, before_update]
        fields:
          text: {type: text, validations: [{name: required}]}
          done: {type: bool, default: false}
    aggregates:
      List:
        audit: true
        fields:
          name: {type: string, validations: [{name: required}]}
          description: {type: text}
        children:
          items:
            of: Item
            audit: true
    api:
      base_path: /todo
      handlers:
        - {id: todo_items_list,   route: "GET /items",        source: repo, model: Item, op: list}
        - {id: todo_items_create, route: "POST /items",       source: repo, model: Item, op: create}
        - {id: todo_items_get,    route: "GET /items/{id}",   source: repo, model: Item, op: get}
        - {id: todo_items_update, route: "PATCH /items/{id}", source: repo, model: Item, op: update}
        - {id: todo_items_delete, route: "DELETE /items/{id}", source: repo, model: Item, op: delete}
```

**Generated Structure:**
```
examples/ref/
├── hatmax.yml                    # Config replicated at monorepo root
└── services/todo/
    ├── go.mod                    # With hm library in dev mode
    ├── config.yaml
    ├── main.go
    ├── Makefile
    └── internal/
        ├── config/
        │   ├── config.go
        │   └── xparams.go
        ├── todo/
        │   ├── item.go           # Model + aggregate root
        │   ├── list.go           # Aggregate with children
        │   ├── itemrepo.go       # Repository interface
        │   ├── itemservice.go    # Service interface
        │   ├── itemhandler.go    # HTTP handlers
        │   └── itemvalidator.go  # Validation functions
        ├── sqlite/
        │   ├── itemrepo.go       # SQLite implementation
        │   └── item_queries.go   # SQL queries
        └── mongo/
            └── itemrepo.go       # MongoDB implementation
```

**Features:**
- Single entity CRUD operations
- DDD aggregate roots with children
- Multiple repository implementations (SQLite, MongoDB)
- Audit fields and lifecycle hooks
- Declarative handlers with automatic inference
- Authentication and authorization
- Validation with custom rules

## Planned Implementation

### `domain` Service

Multi-entity bounded context with relations, complex aggregates, and use cases. Extends atom capabilities with inter-entity relationships and business workflows.

### 3. feature (composite) - API Orchestrator
**What**: Service without database that orchestrates others via API calls
**How**: Defines clients and use cases with declarative steps; handlers call use cases
**Why**: Business logic combining data from multiple services without duplicating repositories

### 4. web - Presentation Layer
**What**: Presentation layer, HTML/HTMX, no own database
**How**: Defines pages and fragments that consume clients to other services
**Why**: Separate UI from business logic, maintain coherent rendering and navigation

## Detailed Service Specifications

### 1. atom Service

**YAML Structure:**
```yaml
version: 0.2
services:
  todo:
    kind: atom
    repo_impl: [sqlite, mongo]  # Available repository implementations

    models:
      Item:
        options:
          audit: true
          lifecycle: [before_create, before_update]
        fields:
          text: {type: text, validations: [{name: required}]}
          done: {type: bool, default: false}

    api:
      base_path: /todo
      handlers:
        - {id: todo_items_list,   route: "GET /items",        source: repo, model: Item, op: list}
        - {id: todo_items_create, route: "POST /items",       source: repo, model: Item, op: create}
        - {id: todo_items_get,    route: "GET /items/{id}",   source: repo, model: Item, op: get}
        - {id: todo_items_update, route: "PATCH /items/{id}", source: repo, model: Item, op: update}
        - {id: todo_items_delete, route: "DELETE /items/{id}", source: repo, model: Item, op: delete}
```

**Generated Structure:**
```
app/services/todo/
├── internal/todo/
│   ├── item.go           # Model with audit fields
│   ├── itemrepo.go       # Repository interface
│   ├── itemhandler.go    # HTTP handlers
│   └── itemvalidator.go  # Validation functions
├── internal/sqlite/
│   ├── itemrepo.go       # SQLite implementation
│   └── item_queries.sql  # SQL queries
├── internal/mongo/
│   └── itemrepo.go       # MongoDB implementation
└── go.mod
```

**Use Cases:**
- User management
- Product catalog
- Simple logging/auditing
- Configuration storage
- Any bounded context with single entity

### 2. feature (with repos) Service

**YAML Structure:**
```yaml
version: 0.1
name: "auth-system"
package: "github.com/company/auth-system"
services:
  auth:
    kind: domain
    repo_impl: [sqlite]

    models:
      User:
        options: {audit: true}
        fields:
          email: {type: email, validations: [required, {name: email}, {name: unique}]}
          name:  {type: string, validations: [required, {min_len: 2}]}
          active: {type: bool, default: true}

      Role:
        fields:
          code: {type: string, validations: [required, {name: unique}]}
          name: {type: string, validations: [required]}

      Permission:
        fields:
          resource: {type: string, validations: [required]}
          action:   {type: string, validations: [required]}

    relations:
      - {from: User, to: Role, kind: many_to_many, via: user_roles}
      - {from: Role, to: Permission, kind: many_to_many, via: role_permissions}

    aggregates:
      user_full:
        source: User
        preload: [roles, roles.permissions]

      role_with_permissions:
        source: Role
        preload: [permissions]

    usecases:
      UserDeactivate:
        input: {user_id: string}
        steps:
          - load: {aggregate: user_full, by: user_id, as: user}
          - validate: {expr: "user.active == true", error: "user_already_inactive"}
          - update: {model: User, set: {active: false}, where: {id: user_id}}
          - emit: {event: "user.deactivated", data: {user_id: user_id}}

      AssignRole:
        input: {user_id: string, role_code: string}
        steps:
          - load: {model: User, by: user_id, as: user}
          - load: {model: Role, by: {code: role_code}, as: role}
          - create: {relation: user_roles, data: {user_id: user_id, role_id: role.id}}

    api:
      base_path: /auth
      handlers:
        # Standard CRUD
        - {route: "GET /users",            source: repo,      model: User, op: list}
        - {route: "POST /users",           source: repo,      model: User, op: create}
        - {route: "GET /users/{id}",       source: repo,      model: User, op: get}

        # Aggregate endpoints
        - {route: "GET /users/{id}/full",  source: aggregate, model: user_full}
        - {route: "GET /roles/{id}/permissions", source: aggregate, model: role_with_permissions}

        # Use case endpoints
        - {route: "PATCH /users/{id}/deactivate", source: usecase, usecase: UserDeactivate}
        - {route: "POST /users/{id}/roles",       source: usecase, usecase: AssignRole}
```

**Generated Structure:**
```
examples/auth-system/
├── hatmax.yml
└── services/auth/
    ├── go.mod
    ├── config.yaml
    ├── main.go
    ├── Makefile
    └── internal/
        ├── config/
        ├── auth/
        │   ├── user.go, role.go, permission.go     # Models
        │   ├── userrepo.go, rolerepo.go, ...       # Repository interfaces
        │   ├── aggregates.go                       # Aggregate loaders
        │   ├── usecases.go                         # Use case implementations
        │   ├── userhandler.go, rolehandler.go      # HTTP handlers
        │   └── validators.go                       # Validation functions
        ├── sqlite/
        │   ├── userrepo.go, rolerepo.go, ...       # Repository implementations
        │   ├── relations.sql                       # Junction table definitions
        │   └── migrations/                         # Database migrations
        └── events/
            └── emitter.go                          # Event emission
```

**Use Cases:**
- Authentication and authorization
- Order management with line items
- Content management with categories/tags
- Inventory with suppliers and locations

### `composite` Service

API orchestrator without database that coordinates multiple services through HTTP calls and declarative workflows.

**YAML Structure:**
```yaml
version: 0.1
name: "billing-system"
package: "github.com/company/billing-system"
services:
  billing:
    kind: composite
    # No repo_impl - orchestrates other services

    clients:
      users:    {base_url: "http://auth:8080"}
      payments: {base_url: "http://payments:8080"}
      notify:   {base_url: "http://notifications:8080"}

    types:
      Invoice:
        fields:
          id: string
          user_id: string
          amount: decimal
          charge_id: string
          issued_at: datetime

      ChargeRequest:
        fields:
          user_id: string
          amount: decimal
          description: string

    usecases:
      IssueInvoice:
        input: {user_id: string, amount: decimal, description: string}
        output: Invoice
        steps:
          - http_get:  {client: users, path: "/auth/users/{user_id}", as: user}
          - validate:  {expr: "user.active == true", error: "user_inactive"}
          - http_post: {client: payments, path: "/payments/charges",
                       body: {user_id: "{{user.id}}", amount: "{{amount}}", description: "{{description}}"},
                       as: charge}
          - transform: {as: invoice, expr: "createInvoice(user, charge, amount)"}
          - http_post: {client: notify, path: "/notifications/email",
                       body: {to: "{{user.email}}", template: "invoice_issued", data: "{{invoice}}"}}
          - return: "{{invoice}}"

      RefundInvoice:
        input: {invoice_id: string, reason: string}
        steps:
          - http_get:  {client: payments, path: "/payments/charges/{charge_id}/refund",
                       body: {reason: "{{reason}}"}, as: refund}
          - http_post: {client: notify, path: "/notifications/email",
                       body: {template: "refund_processed", data: "{{refund}}"}}

    api:
      base_path: /billing
      handlers:
        - {route: "POST /invoices",              source: usecase, usecase: IssueInvoice}
        - {route: "POST /invoices/{id}/refund",  source: usecase, usecase: RefundInvoice}
```

**Generated Structure:**
```
examples/billing-system/
├── hatmax.yml
└── services/billing/
    ├── go.mod
    ├── config.yaml
    ├── main.go
    ├── Makefile
    └── internal/
        ├── config/
        ├── billing/
        │   ├── types.go          # Request/response types
        │   ├── clients.go        # HTTP client wrappers
        │   ├── usecases.go       # Orchestration logic
        │   ├── handlers.go       # HTTP handlers
        │   └── transforms.go     # Data transformation functions
        └── http/
            └── client.go         # Generic HTTP client utilities
```

**Use Cases:**
- Payment processing workflows
- Order fulfillment orchestration
- Multi-service reporting
- Integration with external APIs
- Business process automation

### `web` Service

Presentation layer with HTML templates, HTMX support, and form handling. Consumes other services via HTTP clients.

**YAML Structure:**
```yaml
version: 0.1
name: "portal-system"
package: "github.com/company/portal-system"
services:
  portal:
    kind: web

    clients:
      auth: {base_url: "http://auth:8080"}
      todo: {base_url: "http://todo:8080"}
      billing: {base_url: "http://billing:8080"}

    static:
      css:  "assets/css"
      js:   "assets/js"
      img:  "assets/images"

    pages:
      login:
        route: "GET /login"
        render: {template: "pages/login.html"}

      dashboard:
        route: "GET /dashboard"
        middleware: [auth_required]
        load:
          - http_get: {client: auth, path: "/auth/users/{current_user}/full", as: user}
          - http_get: {client: todo, path: "/todo/items?owner={{user.id}}", as: items}
          - http_get: {client: billing, path: "/billing/invoices?user={{user.id}}", as: invoices}
        render: {template: "pages/dashboard.html",
                data: {user: "{{user}}", items: "{{items}}", invoices: "{{invoices}}"}}

      user_profile:
        route: "GET /profile"
        middleware: [auth_required]
        load:
          - http_get: {client: auth, path: "/auth/users/{current_user}", as: user}
        render: {template: "pages/profile.html", data: {user: "{{user}}"}}

    fragments:
      items_table:
        route: "GET /partials/items"
        middleware: [auth_required]
        params: [owner_id]
        load:
          - http_get: {client: todo, path: "/todo/items?owner={{owner_id}}", as: items}
        render: {template: "partials/items_table.html", data: {items: "{{items}}"}}

      invoice_summary:
        route: "GET /partials/invoices/summary"
        middleware: [auth_required]
        params: [user_id, period]
        load:
          - http_get: {client: billing, path: "/billing/invoices?user={{user_id}}&period={{period}}", as: invoices}
        render: {template: "partials/invoice_summary.html", data: {invoices: "{{invoices}}"}}

    forms:
      create_item:
        route: "POST /items"
        middleware: [auth_required]
        validate: {text: required}
        submit:
          - http_post: {client: todo, path: "/todo/items",
                       body: {text: "{{text}}", owner: "{{current_user}}"}}
        success: {redirect: "/dashboard"}
        error:   {fragment: items_table, with_errors: true}
```

**Generated Structure:**
```
examples/portal-system/
├── hatmax.yml
└── services/portal/
    ├── go.mod
    ├── config.yaml
    ├── main.go
    ├── Makefile
    ├── internal/
    │   ├── config/
    │   └── web/
    │       ├── pages.go          # Page handlers
    │       ├── fragments.go      # Fragment handlers
    │       ├── forms.go          # Form handlers
    │       ├── clients.go        # API client wrappers
    │       └── middleware.go     # Authentication middleware
    ├── templates/
    │   ├── pages/
    │   │   ├── dashboard.html
    │   │   ├── login.html
    │   │   └── profile.html
    │   ├── partials/
    │   │   ├── items_table.html
    │   │   └── invoice_summary.html
    │   └── layouts/
    │       └── base.html
    └── assets/
        ├── css/
        ├── js/
        └── images/
```

**Use Cases:**
- Administrative dashboards
- Customer portals
- Internal tools
- HTMX-powered SPAs
- Traditional multi-page applications

## Service Evolution

```
atom → domain → composite → web
 ↓       ↓         ↓         ↓
single  multiple  orchestr. present.
entity  entities  services  layer
```

### Communication Patterns
- **atom/domain** → Provide HTTP APIs
- **composite** → Consumes atom/domain APIs  
- **web** → Consumes all service types
- All services can emit events for async communication

## Next Steps

### Immediate
1. **Polish atom implementation**
   - Advanced validation patterns
   - Authentication middleware integration
   - Performance optimizations

2. **Begin domain service**
   - Relations and foreign keys
   - Multi-entity aggregates
   - Use case framework

### Medium Term
3. **Complete domain service**
   - Event emission
   - Migration generation
   - Complex business workflows

4. **Start composite service**
   - HTTP client generation
   - Declarative orchestration engine

### Long Term
5. **Web service implementation**
6. **Advanced features** (tracing, monitoring, deployment)

---

**Summary**: Four service types cover the complete backend architecture spectrum. **atom** (currently implemented) handles single-entity CRUD with DDD aggregates support. **domain**, **composite**, and **web** services are planned to extend capabilities for multi-entity bounded contexts, API orchestration, and presentation layers respectively.
