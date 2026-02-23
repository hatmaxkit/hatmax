package pagination

import (
	"testing"
)

func TestNewParams(t *testing.T) {
	tests := []struct {
		name             string
		page             int
		pageSize         int
		expectedPage     int
		expectedPageSize int
	}{
		{
			name:             "valid params",
			page:             2,
			pageSize:         10,
			expectedPage:     2,
			expectedPageSize: 10,
		},
		{
			name:             "zero page defaults to 1",
			page:             0,
			pageSize:         10,
			expectedPage:     1,
			expectedPageSize: 10,
		},
		{
			name:             "negative page defaults to 1",
			page:             -5,
			pageSize:         10,
			expectedPage:     1,
			expectedPageSize: 10,
		},
		{
			name:             "zero pageSize defaults to DefaultPageSize",
			page:             1,
			pageSize:         0,
			expectedPage:     1,
			expectedPageSize: DefaultPageSize,
		},
		{
			name:             "negative pageSize defaults to DefaultPageSize",
			page:             1,
			pageSize:         -10,
			expectedPage:     1,
			expectedPageSize: DefaultPageSize,
		},
		{
			name:             "pageSize exceeding max is capped",
			page:             1,
			pageSize:         500,
			expectedPage:     1,
			expectedPageSize: MaxPageSize,
		},
		{
			name:             "pageSize at max is allowed",
			page:             1,
			pageSize:         MaxPageSize,
			expectedPage:     1,
			expectedPageSize: MaxPageSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := NewParams(tt.page, tt.pageSize)

			if params.Page != tt.expectedPage {
				t.Errorf("Page = %d, want %d", params.Page, tt.expectedPage)
			}

			if params.PageSize != tt.expectedPageSize {
				t.Errorf("PageSize = %d, want %d", params.PageSize, tt.expectedPageSize)
			}
		})
	}
}

func TestParams_Offset(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		expectedOffset int
	}{
		{
			name:           "first page",
			page:           1,
			pageSize:       20,
			expectedOffset: 0,
		},
		{
			name:           "second page",
			page:           2,
			pageSize:       20,
			expectedOffset: 20,
		},
		{
			name:           "third page with size 10",
			page:           3,
			pageSize:       10,
			expectedOffset: 20,
		},
		{
			name:           "fifth page with size 15",
			page:           5,
			pageSize:       15,
			expectedOffset: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := NewParams(tt.page, tt.pageSize)
			offset := params.Offset()

			if offset != tt.expectedOffset {
				t.Errorf("Offset() = %d, want %d", offset, tt.expectedOffset)
			}
		})
	}
}

func TestParams_Limit(t *testing.T) {
	params := NewParams(1, 25)
	if params.Limit() != 25 {
		t.Errorf("Limit() = %d, want 25", params.Limit())
	}
}

func TestNewResult(t *testing.T) {
	tests := []struct {
		name               string
		items              []string
		totalCount         int
		page               int
		pageSize           int
		expectedTotalPages int
		expectedHasMore    bool
	}{
		{
			name:               "single page",
			items:              []string{"a", "b", "c"},
			totalCount:         3,
			page:               1,
			pageSize:           10,
			expectedTotalPages: 1,
			expectedHasMore:    false,
		},
		{
			name:               "first of multiple pages",
			items:              []string{"a", "b", "c", "d", "e"},
			totalCount:         12,
			page:               1,
			pageSize:           5,
			expectedTotalPages: 3,
			expectedHasMore:    true,
		},
		{
			name:               "middle page",
			items:              []string{"f", "g", "h", "i", "j"},
			totalCount:         12,
			page:               2,
			pageSize:           5,
			expectedTotalPages: 3,
			expectedHasMore:    true,
		},
		{
			name:               "last page",
			items:              []string{"k", "l"},
			totalCount:         12,
			page:               3,
			pageSize:           5,
			expectedTotalPages: 3,
			expectedHasMore:    false,
		},
		{
			name:               "empty result",
			items:              []string{},
			totalCount:         0,
			page:               1,
			pageSize:           10,
			expectedTotalPages: 0,
			expectedHasMore:    false,
		},
		{
			name:               "exact page boundary",
			items:              []string{"a", "b", "c", "d", "e"},
			totalCount:         10,
			page:               1,
			pageSize:           5,
			expectedTotalPages: 2,
			expectedHasMore:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := NewParams(tt.page, tt.pageSize)
			result := NewResult(tt.items, tt.totalCount, params)

			if result.TotalPages != tt.expectedTotalPages {
				t.Errorf("TotalPages = %d, want %d", result.TotalPages, tt.expectedTotalPages)
			}

			if result.HasMore != tt.expectedHasMore {
				t.Errorf("HasMore = %v, want %v", result.HasMore, tt.expectedHasMore)
			}

			if result.TotalCount != tt.totalCount {
				t.Errorf("TotalCount = %d, want %d", result.TotalCount, tt.totalCount)
			}

			if result.Page != tt.page {
				t.Errorf("Page = %d, want %d", result.Page, tt.page)
			}

			if result.PageSize != tt.pageSize {
				t.Errorf("PageSize = %d, want %d", result.PageSize, tt.pageSize)
			}

			if len(result.Items) != len(tt.items) {
				t.Errorf("len(Items) = %d, want %d", len(result.Items), len(tt.items))
			}
		})
	}
}

func TestResult_HasPrevious(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		expected bool
	}{
		{"first page", 1, false},
		{"second page", 2, true},
		{"tenth page", 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := NewParams(tt.page, 10)
			result := NewResult([]string{"a"}, 100, params)

			if result.HasPrevious() != tt.expected {
				t.Errorf("HasPrevious() = %v, want %v", result.HasPrevious(), tt.expected)
			}
		})
	}
}

func TestResult_PreviousPage(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		expected int
	}{
		{"first page returns 1", 1, 1},
		{"second page returns 1", 2, 1},
		{"fifth page returns 4", 5, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := NewParams(tt.page, 10)
			result := NewResult([]string{"a"}, 100, params)

			if result.PreviousPage() != tt.expected {
				t.Errorf("PreviousPage() = %d, want %d", result.PreviousPage(), tt.expected)
			}
		})
	}
}

func TestResult_NextPage(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		totalCount int
		pageSize   int
		expected   int
	}{
		{"has more pages", 1, 30, 10, 2},
		{"last page", 3, 30, 10, 3},
		{"middle page", 2, 30, 10, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := NewParams(tt.page, tt.pageSize)
			result := NewResult([]string{"a"}, tt.totalCount, params)

			if result.NextPage() != tt.expected {
				t.Errorf("NextPage() = %d, want %d", result.NextPage(), tt.expected)
			}
		})
	}
}

func TestResult_IsEmpty(t *testing.T) {
	params := NewParams(1, 10)

	emptyResult := NewResult([]string{}, 0, params)
	if !emptyResult.IsEmpty() {
		t.Error("IsEmpty() should return true for empty items")
	}

	nonEmptyResult := NewResult([]string{"a"}, 1, params)
	if nonEmptyResult.IsEmpty() {
		t.Error("IsEmpty() should return false for non-empty items")
	}
}

func TestResult_StartIndex(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		items    []string
		expected int
	}{
		{"first page", 1, 10, []string{"a", "b"}, 1},
		{"second page", 2, 10, []string{"a", "b"}, 11},
		{"third page size 5", 3, 5, []string{"a"}, 11},
		{"empty result", 1, 10, []string{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := NewParams(tt.page, tt.pageSize)
			result := NewResult(tt.items, 100, params)

			if result.StartIndex() != tt.expected {
				t.Errorf("StartIndex() = %d, want %d", result.StartIndex(), tt.expected)
			}
		})
	}
}

func TestResult_EndIndex(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		pageSize   int
		totalCount int
		items      []string
		expected   int
	}{
		{"full first page", 1, 10, 25, []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, 10},
		{"full second page", 2, 10, 25, []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, 20},
		{"partial last page", 3, 10, 25, []string{"a", "b", "c", "d", "e"}, 25},
		{"empty result", 1, 10, 0, []string{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := NewParams(tt.page, tt.pageSize)
			result := NewResult(tt.items, tt.totalCount, params)

			if result.EndIndex() != tt.expected {
				t.Errorf("EndIndex() = %d, want %d", result.EndIndex(), tt.expected)
			}
		})
	}
}
