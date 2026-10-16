// Package table renders values as an aligned text table, one row per value.
package table

import (
	"io"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/spoolersh/spooler-cli/internal/format"
)

var _ format.ColumnsEncoding[int] = (*Encoding[int])(nil)

// Encoding renders each value of T as one row of an aligned table, under a
// header row naming the columns in upper case ("stats.visible" as
// STATS_VISIBLE). A value the json format omits prints as a dash.
//
// The zero value prints every leaf field of T.
type Encoding[T any] struct {
	// Human renders the fields tagged `human` in their human form, as
	// format.Columns.Row does with human set.
	Human bool

	columns *format.Columns[T]
}

// SetColumns implements format.ColumnsEncoding.
func (e *Encoding[T]) SetColumns(c *format.Columns[T]) {
	e.columns = c
}

// EncoderTo implements format.Encoding.
func (e *Encoding[T]) EncoderTo(w io.Writer) format.Encoder[T] {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	headers := e.columns.Paths()
	for i, path := range headers {
		headers[i] = header(path)
	}
	writeRow(tw, headers)
	return &encoder[T]{
		columns: e.columns,
		human:   e.Human,
		tw:      tw,
		cells:   make([]string, len(headers)),
	}
}

type encoder[T any] struct {
	columns *format.Columns[T]
	human   bool
	tw      *tabwriter.Writer
	cells   []string
}

func (e *encoder[T]) Encode(v T) error {
	row, err := e.columns.Row(v, e.human)
	if err != nil {
		return err
	}
	for i, c := range row {
		if c.Omitted {
			e.cells[i] = "-"
		} else {
			e.cells[i] = c.Text
		}
	}
	writeRow(e.tw, e.cells)
	return nil
}

func (e *encoder[T]) Flush() error {
	return e.tw.Flush()
}

func writeRow(w io.Writer, cells []string) {
	// The last cell is not tab-terminated, or tabwriter would pad it and
	// leave trailing whitespace on every row.
	io.WriteString(w, strings.Join(cells, "\t"))
	io.WriteString(w, "\n")
}

// header names the column of the field at a json path: the path in upper
// case, "stats.visible" as STATS_VISIBLE.
func header(path string) string {
	return toUppercase(strings.ReplaceAll(path, ".", "_"))
}

// toUppercase converts a camelCase JSON name to an UPPER_SNAKE_CASE header.
//
// Examples:
//
//	createdAt -> CREATED_AT
//	maxHTTPRetries -> MAX_HTTP_RETRIES
func toUppercase(name string) string {
	var sb strings.Builder
	sb.Grow(len(name) + 4)
	rs := []rune(name)
	for i, r := range rs {
		if i > 0 && unicode.IsUpper(r) {
			prev := rs[i-1]
			// A word starts at an upper after a lower or a digit (createdAt),
			// or at the last upper of an acronym run (HTTPRetries).
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			switch {
			case unicode.IsLower(prev), unicode.IsDigit(prev):
				sb.WriteByte('_')
			case unicode.IsUpper(prev) && nextLower:
				sb.WriteByte('_')
			}
		}
		sb.WriteRune(unicode.ToUpper(r))
	}
	return sb.String()
}
