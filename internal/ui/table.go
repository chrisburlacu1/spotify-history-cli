package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type TableAlignment int

const (
	AlignLeft TableAlignment = iota
	AlignRight
	AlignCenter
)

type Table struct {
	Headers    []string
	Alignments []TableAlignment
	Rows       [][]string
}

func NewTable(headers ...string) *Table {
	alignments := make([]TableAlignment, len(headers))
	for i := range alignments {
		alignments[i] = AlignLeft
	}
	return &Table{
		Headers:    headers,
		Alignments: alignments,
		Rows:       [][]string{},
	}
}

func (t *Table) SetAlignment(colIndex int, align TableAlignment) *Table {
	if colIndex >= 0 && colIndex < len(t.Alignments) {
		t.Alignments[colIndex] = align
	}
	return t
}

func (t *Table) AddRow(cols ...interface{}) *Table {
	row := make([]string, len(cols))
	for i, c := range cols {
		row[i] = fmt.Sprintf("%v", c)
	}
	t.Rows = append(t.Rows, row)
	return t
}

func (t *Table) Render() string {
	if len(t.Headers) == 0 && len(t.Rows) == 0 {
		return "No data."
	}

	numCols := len(t.Headers)
	colWidths := make([]int, numCols)

	for i, h := range t.Headers {
		w := utf8.RuneCountInString(h)
		if w > colWidths[i] {
			colWidths[i] = w
		}
	}

	for _, row := range t.Rows {
		for i, cell := range row {
			if i < numCols {
				w := utf8.RuneCountInString(cell)
				if w > colWidths[i] {
					colWidths[i] = w
				}
			}
		}
	}

	var sb strings.Builder

	// Header row
	var headerCells []string
	for i, h := range t.Headers {
		headerCells = append(headerCells, Pad(h, colWidths[i], t.Alignments[i]))
	}
	sb.WriteString(Bold + Cyan + strings.Join(headerCells, " │ ") + Reset + "\n")

	// Separator
	var sepCells []string
	for _, w := range colWidths {
		sepCells = append(sepCells, strings.Repeat("─", w))
	}
	sb.WriteString(Gray + strings.Join(sepCells, "─┼─") + Reset + "\n")

	// Data rows
	for _, row := range t.Rows {
		var cells []string
		for i := 0; i < numCols; i++ {
			val := ""
			if i < len(row) {
				val = row[i]
			}
			align := AlignLeft
			if i < len(t.Alignments) {
				align = t.Alignments[i]
			}
			cells = append(cells, Pad(val, colWidths[i], align))
		}
		sb.WriteString(strings.Join(cells, " │ ") + "\n")
	}

	return sb.String()
}

func Pad(s string, width int, align TableAlignment) string {
	length := utf8.RuneCountInString(s)
	if length >= width {
		return s
	}
	diff := width - length
	switch align {
	case AlignRight:
		return strings.Repeat(" ", diff) + s
	case AlignCenter:
		left := diff / 2
		right := diff - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	default:
		return s + strings.Repeat(" ", diff)
	}
}
