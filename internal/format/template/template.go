// Package template renders values through a Go text/template over their
// json-named fields, one line per value.
package template

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/spoolersh/spooler-cli/internal/format"
	"github.com/spoolersh/spooler-cli/internal/format/internal/fields"
)

var _ format.Encoding[int] = (*Encoding[int])(nil)

// Encoding renders each value of T by executing a template against the
// value's fields as nested maps by their json names, so the template reads
// {{.id}} and {{.stats.visible}}, and appends a newline. Each field is its
// Go value as is: a number compares and prints as one, and a time is a
// time.Time with its methods. A field the value omits is nil, which prints
// as "<no value>" and is false in a condition; a name that is no field of
// T is an execution error naming the fields.
type Encoding[T any] struct {
	text string
	tmpl *template.Template
}

// Parse parses text as a text/template with no functions beyond the
// package's own.
func Parse[T any](text string) (*Encoding[T], error) {
	tmpl, err := template.New("--template").
		Option("missingkey=error").
		Parse(text)
	if err != nil {
		return nil, err
	}
	return &Encoding[T]{
		text: text,
		tmpl: tmpl,
	}, nil
}

// Text returns the template as parsed.
func (e *Encoding[T]) Text() string {
	return e.text
}

// EncoderTo implements format.Encoding.
func (e *Encoding[T]) EncoderTo(w io.Writer) format.Encoder[T] {
	return &encoder[T]{
		tmpl: e.tmpl,
		w:    w,
	}
}

type encoder[T any] struct {
	tmpl *template.Template
	w    io.Writer
	buf  bytes.Buffer
}

// Encode renders v into a buffer first, so a failing template writes nothing.
func (e *encoder[T]) Encode(v T) error {
	e.buf.Reset()
	if err := e.tmpl.Execute(&e.buf, form(v)); err != nil {
		return withFields[T](err)
	}
	e.buf.WriteByte('\n')
	_, err := e.w.Write(e.buf.Bytes())
	return err
}

func (e *encoder[T]) Flush() error {
	return nil
}

// withFields appends the fields of T to an execution error for a name none
// of them has, the one key the data can lack; any other error is returned
// as it is, its message already naming the template and the position.
func withFields[T any](err error) error {
	var exec template.ExecError
	if !errors.As(err, &exec) ||
		!strings.Contains(err.Error(), "map has no entry for key") {
		return err
	}
	return fmt.Errorf(
		"%w; fields: %s",
		err, strings.Join(format.PathsFor[T](), ", "),
	)
}

// form returns v as nested maps keyed by json names, through the walker the
// other formats render through, so a template sees the same fields: each
// leaf its Go value as is, and one its json tag omits nil, so that every
// name of T is a key and only a name that is no field is missing.
func form[T any](v T) map[string]any {
	data := make(map[string]any)
	for _, f := range fields.Of(v) {
		m := data
		path := strings.Split(f.Path, ".")
		for _, name := range path[:len(path)-1] {
			next, ok := m[name].(map[string]any)
			if !ok {
				next = make(map[string]any)
				m[name] = next
			}
			m = next
		}
		if fields.Omitted(f) {
			m[path[len(path)-1]] = nil
			continue
		}
		m[path[len(path)-1]] = f.Value.Interface()
	}
	return data
}
