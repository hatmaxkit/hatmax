package kit

import (
	"html/template"
	"strings"
	"testing"
)

func TestTableRender(t *testing.T) {
	tests := []struct {
		name     string
		table    *Table
		contains []string
	}{
		{
			name:     "empty table",
			table:    NewTable(),
			contains: []string{`class="table"`, `No data available`},
		},
		{
			name:     "table with columns",
			table:    NewTable().Columns(Col("name", "Name"), Col("email", "Email")),
			contains: []string{`<thead>`, `<th`, `Name`, `Email`},
		},
		{
			name:     "striped table",
			table:    NewTable().Striped(),
			contains: []string{`table--striped`},
		},
		{
			name:     "hoverable table",
			table:    NewTable().Hoverable(),
			contains: []string{`table--hoverable`},
		},
		{
			name:     "compact table",
			table:    NewTable().Compact(),
			contains: []string{`table--compact`},
		},
		{
			name:     "custom empty text",
			table:    NewTable().EmptyText("Nothing here"),
			contains: []string{`Nothing here`},
		},
		{
			name:     "table with id",
			table:    NewTable().ID("my-table"),
			contains: []string{`id="my-table"`},
		},
		{
			name:     "table with class",
			table:    NewTable().Class("custom-table"),
			contains: []string{`custom-table`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.table.Render())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Render() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestTableWithRows(t *testing.T) {
	table := NewTable().
		Columns(Col("name", "Name"), Col("age", "Age")).
		Rows(
			NewRow(Text("Alice"), Text("30")),
			NewRow(Text("Bob"), Text("25")),
		)

	html := string(table.Render())

	if !strings.Contains(html, "Alice") {
		t.Error("Render() missing Alice")
	}
	if !strings.Contains(html, "Bob") {
		t.Error("Render() missing Bob")
	}
	if !strings.Contains(html, "30") {
		t.Error("Render() missing 30")
	}
}

func TestTableAddRow(t *testing.T) {
	table := NewTable().
		Columns(Col("name", "Name")).
		AddRow(NewRow(Text("Test")))

	html := string(table.Render())

	if !strings.Contains(html, "Test") {
		t.Error("AddRow() row not added")
	}
}

func TestColumnOptions(t *testing.T) {
	col := Col("test", "Test").WithWidth("100px").WithAlign("right").WithClass("custom")

	if col.Width != "100px" {
		t.Errorf("WithWidth() = %q, want %q", col.Width, "100px")
	}
	if col.Align != "right" {
		t.Errorf("WithAlign() = %q, want %q", col.Align, "right")
	}
	if col.Class != "custom" {
		t.Errorf("WithClass() = %q, want %q", col.Class, "custom")
	}
}

func TestRowOptions(t *testing.T) {
	row := NewRow(Text("Test")).WithID("row-1").WithClass("highlight")

	if row.ID != "row-1" {
		t.Errorf("WithID() = %q, want %q", row.ID, "row-1")
	}
	if row.Class != "highlight" {
		t.Errorf("WithClass() = %q, want %q", row.Class, "highlight")
	}
}

func TestCellTypes(t *testing.T) {
	textCell := Text("Hello")
	if string(textCell.Content) != "Hello" {
		t.Error("Text() content not set")
	}

	htmlCell := HTML(template.HTML("<strong>Bold</strong>"))
	if string(htmlCell.Content) != "<strong>Bold</strong>" {
		t.Error("HTML() content not set")
	}

	classCell := CellWithClass(template.HTML("Test"), "custom")
	if classCell.Class != "custom" {
		t.Error("CellWithClass() class not set")
	}
}

func TestTextHTMLEscaping(t *testing.T) {
	cell := Text("<script>alert('xss')</script>")
	if strings.Contains(string(cell.Content), "<script>") {
		t.Error("Text() did not escape HTML")
	}
}

func TestTableColumnRender(t *testing.T) {
	table := NewTable().
		Columns(
			Col("name", "Name").WithWidth("200px"),
			Col("actions", "").WithAlign("right"),
		)

	html := string(table.Render())

	if !strings.Contains(html, `style="width: 200px"`) {
		t.Error("Render() missing column width")
	}
	if !strings.Contains(html, `table__th--right`) {
		t.Error("Render() missing column align class")
	}
}

func TestTableRowRender(t *testing.T) {
	table := NewTable().
		Columns(Col("name", "Name")).
		Rows(NewRow(Text("Test")).WithID("row-1").WithClass("selected"))

	html := string(table.Render())

	if !strings.Contains(html, `id="row-1"`) {
		t.Error("Render() missing row id")
	}
	if !strings.Contains(html, `selected`) {
		t.Error("Render() missing row class")
	}
}
