// Package format renders values of one type to a writer, one value at a time.
package format

import (
	"errors"
	"io"
)

// Encoding is an output format for values of T.
type Encoding[T any] interface {
	// EncoderTo returns an Encoder writing values of T to w.
	EncoderTo(w io.Writer) Encoder[T]
}

// Encoder writes values of T in one Encoding.
type Encoder[T any] interface {
	// Encode writes one value.
	Encode(T) error

	// Flush completes the output and must be called once after the last
	// value, after a failure to fetch the next one too: a format that
	// buffers (csv, table) then writes every value encoded so far, whole.
	Flush() error
}

// ColumnsEncoding is an Encoding that lays values out flat, in columns that
// can be selected.
type ColumnsEncoding[T any] interface {
	Encoding[T]

	// SetColumns selects and orders the columns; nil means every leaf
	// field of T.
	SetColumns(*Columns[T])
}

// ErrNoColumns is what SetColumns returns for an Encoding without columns.
var ErrNoColumns = errors.New("output format has no columns")

// SetColumns selects the columns of enc, or returns ErrNoColumns when its
// format has none.
func SetColumns[T any](enc Encoding[T], c *Columns[T]) error {
	ce, ok := enc.(ColumnsEncoding[T])
	if !ok {
		return ErrNoColumns
	}
	ce.SetColumns(c)
	return nil
}
