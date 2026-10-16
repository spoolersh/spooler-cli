// Package text renders values as records: a "key: value" line per field,
// ended by an empty line.
package text

import (
	"io"
	"strings"

	"github.com/spoolersh/spooler-cli/internal/format"
)

var _ format.ColumnsEncoding[int] = (*Encoding[int])(nil)

// Encoding renders each value of T as one record: a line per field with the
// field's json path as the key ("stats.visible: 12"), then an empty line. A
// value the json format omits has no line.
//
// The zero value prints every leaf field of T, unstyled.
type Encoding[T any] struct {
	// KeyStyle is an SGR sequence ("\x1b[38;5;227m") wrapped around each
	// key, reset after it. Empty means plain keys.
	KeyStyle string

	// PathStyle is an SGR sequence wrapped around the path of a nested key,
	// the "stats." of "stats.visible", in place of KeyStyle, so the eye
	// groups the leaves while the line stays flat. Empty styles the whole
	// key with KeyStyle; ignored when KeyStyle is empty.
	PathStyle string

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
	return &encoder[T]{
		columns:   e.columns,
		paths:     e.columns.Paths(),
		style:     e.KeyStyle,
		pathStyle: e.PathStyle,
		human:     e.Human,
		w:         w,
	}
}

type encoder[T any] struct {
	columns   *format.Columns[T]
	paths     []string
	style     string
	pathStyle string
	human     bool
	w         io.Writer
}

// reset ends an SGR style.
const reset = "\x1b[0m"

// Encode writes one record in a single write, so it reaches w whole or not
// at all.
func (e *encoder[T]) Encode(v T) error {
	row, err := e.columns.Row(v, e.human)
	if err != nil {
		return err
	}
	var sb strings.Builder
	for i, c := range row {
		if c.Omitted {
			continue
		}
		key := e.paths[i]
		if e.style != "" {
			if dot := strings.LastIndexByte(key, '.'); dot >= 0 && e.pathStyle != "" {
				sb.WriteString(e.pathStyle)
				sb.WriteString(key[:dot+1])
				sb.WriteString(reset)
				key = key[dot+1:]
			}
			sb.WriteString(e.style)
			sb.WriteString(key)
			sb.WriteString(reset)
		} else {
			sb.WriteString(key)
		}
		sb.WriteString(": ")
		sb.WriteString(c.Text)
		sb.WriteByte('\n')
	}
	sb.WriteByte('\n')
	_, err = io.WriteString(e.w, sb.String())
	return err
}

func (e *encoder[T]) Flush() error {
	return nil
}
