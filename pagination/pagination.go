// Package pagination provides generic pagination utilities.
package pagination

// Params holds pagination parameters from a request.
type Params struct {
	Page     int
	PageSize int
}

// Result holds a paginated result set.
type Result[T any] struct {
	Items      []T  `json:"items"`
	TotalCount int  `json:"total_count"`
	Page       int  `json:"page"`
	PageSize   int  `json:"page_size"`
	TotalPages int  `json:"total_pages"`
	HasMore    bool `json:"has_more"`
}

// DefaultPageSize is the default number of items per page.
const DefaultPageSize = 20

// MaxPageSize is the maximum allowed page size.
const MaxPageSize = 100

// NewParams creates pagination params with defaults applied.
func NewParams(page, pageSize int) Params {
	if page < 1 {
		page = 1
	}

	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}

	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	return Params{
		Page:     page,
		PageSize: pageSize,
	}
}

// Offset calculates the SQL offset for the current page.
func (p Params) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the page size (alias for clarity in SQL contexts).
func (p Params) Limit() int {
	return p.PageSize
}

// NewResult creates a paginated result from items and total count.
func NewResult[T any](items []T, totalCount int, params Params) *Result[T] {
	totalPages := totalCount / params.PageSize
	if totalCount%params.PageSize > 0 {
		totalPages++
	}

	return &Result[T]{
		Items:      items,
		TotalCount: totalCount,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
		HasMore:    params.Page < totalPages,
	}
}

// HasPrevious returns true if there's a previous page.
func (r *Result[T]) HasPrevious() bool {
	return r.Page > 1
}

// PreviousPage returns the previous page number, or 1 if on first page.
func (r *Result[T]) PreviousPage() int {
	if r.Page > 1 {
		return r.Page - 1
	}

	return 1
}

// NextPage returns the next page number, or current if on last page.
func (r *Result[T]) NextPage() int {
	if r.HasMore {
		return r.Page + 1
	}

	return r.Page
}

// IsEmpty returns true if there are no items.
func (r *Result[T]) IsEmpty() bool {
	return len(r.Items) == 0
}

// StartIndex returns the 1-based index of the first item on this page.
func (r *Result[T]) StartIndex() int {
	if r.IsEmpty() {
		return 0
	}

	return (r.Page-1)*r.PageSize + 1
}

// EndIndex returns the 1-based index of the last item on this page.
func (r *Result[T]) EndIndex() int {
	if r.IsEmpty() {
		return 0
	}

	end := r.Page * r.PageSize
	if end > r.TotalCount {
		end = r.TotalCount
	}

	return end
}
