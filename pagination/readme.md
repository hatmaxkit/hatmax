# pagination

Generic pagination for lists.

## Usage

```go
// Parse from request
params := pagination.NewParams(page, pageSize)

// Use in query
rows := db.Query("SELECT ... LIMIT $1 OFFSET $2", params.Limit(), params.Offset())

// Build result
result := pagination.NewResult(items, totalCount, params)

// In templates
{{range .Items}} ... {{end}}
{{if .HasMore}} <a href="?page={{.NextPage}}">Next</a> {{end}}
{{.StartIndex}} - {{.EndIndex}} of {{.TotalCount}}
```

Defaults: page size 20, max 100.
