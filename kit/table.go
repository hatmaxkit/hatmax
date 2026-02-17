package kit

import (
	"fmt"
	"html/template"
	"strings"
)

// Table renders a data table.
type Table struct {
	columns   []Column
	rows      []Row
	striped   bool
	hoverable bool
	compact   bool
	emptyText string
	class     string
	id        string
}

// Column defines a table column.
type Column struct {
	Key   string
	Label string
	Class string
	Width string
	Align string
}

// Row defines a table row.
type Row struct {
	ID    string
	Cells []Cell
	Class string
}

// Cell defines a table cell.
type Cell struct {
	Content template.HTML
	Class   string
}

// NewTable creates a new table.
func NewTable() *Table {
	return &Table{
		emptyText: "No data available",
	}
}

// Columns sets the table columns.
func (t *Table) Columns(cols ...Column) *Table {
	t.columns = cols
	return t
}

// Rows sets the table rows.
func (t *Table) Rows(rows ...Row) *Table {
	t.rows = rows
	return t
}

// AddRow adds a row to the table.
func (t *Table) AddRow(row Row) *Table {
	t.rows = append(t.rows, row)
	return t
}

// Striped enables striped rows.
func (t *Table) Striped() *Table {
	t.striped = true
	return t
}

// Hoverable enables hover effect on rows.
func (t *Table) Hoverable() *Table {
	t.hoverable = true
	return t
}

// Compact enables compact mode.
func (t *Table) Compact() *Table {
	t.compact = true
	return t
}

// EmptyText sets the text shown when table is empty.
func (t *Table) EmptyText(text string) *Table {
	t.emptyText = text
	return t
}

// Class adds custom CSS classes.
func (t *Table) Class(class string) *Table {
	t.class = class
	return t
}

// ID sets the table ID.
func (t *Table) ID(id string) *Table {
	t.id = id
	return t
}

// Render renders the table to HTML.
func (t *Table) Render() template.HTML {
	var classes []string
	classes = append(classes, "table")
	if t.striped {
		classes = append(classes, "table--striped")
	}
	if t.hoverable {
		classes = append(classes, "table--hoverable")
	}
	if t.compact {
		classes = append(classes, "table--compact")
	}
	if t.class != "" {
		classes = append(classes, t.class)
	}

	var html strings.Builder

	// Table wrapper
	html.WriteString(`<div class="table-wrapper">`)

	// Table tag
	fmt.Fprintf(&html, `<table class="%s"`, strings.Join(classes, " "))
	if t.id != "" {
		fmt.Fprintf(&html, ` id="%s"`, template.HTMLEscapeString(t.id))
	}
	html.WriteString(`>`)

	// Header
	if len(t.columns) > 0 {
		html.WriteString(`<thead><tr>`)
		for _, col := range t.columns {
			thClass := "table__th"
			if col.Class != "" {
				thClass += " " + col.Class
			}
			if col.Align != "" {
				thClass += " table__th--" + col.Align
			}

			fmt.Fprintf(&html, `<th class="%s"`, thClass)
			if col.Width != "" {
				fmt.Fprintf(&html, ` style="width: %s"`, template.HTMLEscapeString(col.Width))
			}
			fmt.Fprintf(&html, `>%s</th>`, template.HTMLEscapeString(col.Label))
		}
		html.WriteString(`</tr></thead>`)
	}

	// Body
	html.WriteString(`<tbody>`)
	if len(t.rows) == 0 {
		colspan := len(t.columns)
		if colspan == 0 {
			colspan = 1
		}
		fmt.Fprintf(&html, `<tr><td colspan="%d" class="table__empty">%s</td></tr>`,
			colspan, template.HTMLEscapeString(t.emptyText))
	} else {
		for _, row := range t.rows {
			trClass := "table__tr"
			if row.Class != "" {
				trClass += " " + row.Class
			}
			fmt.Fprintf(&html, `<tr class="%s"`, trClass)
			if row.ID != "" {
				fmt.Fprintf(&html, ` id="%s"`, template.HTMLEscapeString(row.ID))
			}
			html.WriteString(`>`)

			for _, cell := range row.Cells {
				tdClass := "table__td"
				if cell.Class != "" {
					tdClass += " " + cell.Class
				}
				fmt.Fprintf(&html, `<td class="%s">%s</td>`, tdClass, cell.Content)
			}

			html.WriteString(`</tr>`)
		}
	}
	html.WriteString(`</tbody>`)

	html.WriteString(`</table>`)
	html.WriteString(`</div>`)

	return template.HTML(html.String())
}

// Col creates a new column.
func Col(key, label string) Column {
	return Column{Key: key, Label: label}
}

// WithWidth sets the column width.
func (c Column) WithWidth(width string) Column {
	c.Width = width
	return c
}

// WithAlign sets the column alignment.
func (c Column) WithAlign(align string) Column {
	c.Align = align
	return c
}

// WithClass sets the column class.
func (c Column) WithClass(class string) Column {
	c.Class = class
	return c
}

// NewRow creates a new row.
func NewRow(cells ...Cell) Row {
	return Row{Cells: cells}
}

// WithID sets the row ID.
func (r Row) WithID(id string) Row {
	r.ID = id
	return r
}

// WithClass sets the row class.
func (r Row) WithClass(class string) Row {
	r.Class = class
	return r
}

// Text creates a cell with text content.
func Text(s string) Cell {
	return Cell{Content: template.HTML(template.HTMLEscapeString(s))}
}

// HTML creates a cell with HTML content.
func HTML(h template.HTML) Cell {
	return Cell{Content: h}
}

// CellWithClass creates a cell with custom class.
func CellWithClass(content template.HTML, class string) Cell {
	return Cell{Content: content, Class: class}
}
