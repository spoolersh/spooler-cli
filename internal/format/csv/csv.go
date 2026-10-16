// Package csv renders values as CSV (RFC 4180), one record per value.
package csv

import (
	"encoding/csv"
	"io"

	"github.com/spoolersh/spooler-cli/internal/format"
)

var _ format.ColumnsEncoding[int] = (*Encoding[int])(nil)

// Encoding renders each value of T as one CSV record, under a header record
// of the columns' json paths. A value the json format omits is an empty
// field.
//
// The zero value prints every leaf field of T.
type Encoding[T any] struct {
	columns *format.Columns[T]
}

// SetColumns implements format.ColumnsEncoding.
func (e *Encoding[T]) SetColumns(c *format.Columns[T]) {
	e.columns = c
}

// EncoderTo implements format.Encoding.
func (e *Encoding[T]) EncoderTo(w io.Writer) format.Encoder[T] {
	cw := csv.NewWriter(w)
	headers := e.columns.Paths()
	cw.Write(headers)
	return &encoder[T]{
		columns: e.columns,
		cw:      cw,
		record:  make([]string, len(headers)),
	}
}

type encoder[T any] struct {
	columns *format.Columns[T]
	cw      *csv.Writer
	record  []string
}

func (e *encoder[T]) Encode(v T) error {
	row, err := e.columns.Row(v, false)
	if err != nil {
		return err
	}
	for i, c := range row {
		e.record[i] = c.Text
	}
	return e.cw.Write(e.record)
}

func (e *encoder[T]) Flush() error {
	e.cw.Flush()
	return e.cw.Error()
}
