// Package json renders values as NDJSON: one JSON document per line.
package json

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/spoolersh/spooler-cli/internal/format"
)

var _ format.Encoding[int] = (*Encoding[int])(nil)

// Encoding renders each value of T as one JSON document on its own line.
type Encoding[T any] struct {
	// Indent, when not empty, is the indentation of each nested level and
	// spreads every document over lines, for a reader at a terminal. Empty
	// keeps each document on one line.
	Indent string

	// KeyStyle is an SGR sequence ("\x1b[38;5;75m") wrapped around each
	// object key, reset after it. Empty means plain keys.
	KeyStyle string

	// StringStyle is an SGR sequence wrapped around each string value,
	// reset after it. Empty means plain strings.
	StringStyle string
}

// EncoderTo implements format.Encoding.
func (e *Encoding[T]) EncoderTo(w io.Writer) format.Encoder[T] {
	if e.KeyStyle != "" || e.StringStyle != "" {
		return &styledEncoder[T]{
			w:           w,
			indent:      e.Indent,
			keyStyle:    e.KeyStyle,
			stringStyle: e.StringStyle,
		}
	}
	enc := json.NewEncoder(w)
	if e.Indent != "" {
		enc.SetIndent("", e.Indent)
	}
	return &encoder[T]{
		enc: enc,
	}
}

type encoder[T any] struct {
	enc *json.Encoder
}

func (e *encoder[T]) Encode(v T) error {
	return e.enc.Encode(v)
}

func (e *encoder[T]) Flush() error {
	return nil
}

// reset ends an SGR style.
const reset = "\x1b[0m"

// styledEncoder writes each document token by token, so a style can wrap a
// token, laid out exactly as encoding/json lays it out with the same indent.
type styledEncoder[T any] struct {
	w           io.Writer
	indent      string
	keyStyle    string
	stringStyle string
}

// Encode writes one record in a single write, so it reaches w whole or not
// at all.
func (e *styledEncoder[T]) Encode(v T) error {
	bts, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := e.restyle(&buf, bts); err != nil {
		return err
	}
	buf.WriteByte('\n')
	_, err = e.w.Write(buf.Bytes())
	return err
}

func (e *styledEncoder[T]) Flush() error {
	return nil
}

// frame is an open object or array: how many members it holds so far, and
// whether the next string in an object is a key.
type frame struct {
	object bool
	n      int
	key    bool
}

// restyle writes the document src to buf with the styles applied.
func (e *styledEncoder[T]) restyle(buf *bytes.Buffer, src []byte) error {
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.UseNumber()
	var stack []frame
	// member ends a value in the innermost open frame.
	member := func() {
		if n := len(stack); n > 0 {
			top := &stack[n-1]
			top.n++
			top.key = top.object
		}
	}
	// newline breaks the line and indents to depth, when indenting.
	newline := func(depth int) {
		if e.indent == "" {
			return
		}
		buf.WriteByte('\n')
		for range depth {
			buf.WriteString(e.indent)
		}
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if d, ok := tok.(json.Delim); ok && (d == '}' || d == ']') {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if top.n > 0 {
				newline(len(stack))
			}
			buf.WriteString(d.String())
			member()
			continue
		}
		if n := len(stack); n > 0 {
			top := &stack[n-1]
			if top.n > 0 && (top.key || !top.object) {
				buf.WriteByte(',')
			}
			if top.key || !top.object {
				newline(n)
			}
			if top.key {
				e.writeString(buf, tok.(string), e.keyStyle)
				buf.WriteByte(':')
				if e.indent != "" {
					buf.WriteByte(' ')
				}
				top.key = false
				continue
			}
		}
		switch v := tok.(type) {
		case json.Delim:
			buf.WriteString(v.String())
			stack = append(stack, frame{
				object: v == '{',
				key:    v == '{',
			})
			continue
		case string:
			e.writeString(buf, v, e.stringStyle)
		case json.Number:
			buf.WriteString(v.String())
		case bool:
			buf.WriteString(strconv.FormatBool(v))
		case nil:
			buf.WriteString("null")
		}
		member()
	}
}

// writeString writes s as a JSON string wrapped in style, if any.
func (e *styledEncoder[T]) writeString(buf *bytes.Buffer, s, style string) {
	bts, _ := json.Marshal(s)
	if style == "" {
		buf.Write(bts)
		return
	}
	buf.WriteString(style)
	buf.Write(bts)
	buf.WriteString(reset)
}

// Strip removes every SGR sequence from s. It is for tests and callers that
// compare styled output with plain.
func Strip(s string) string {
	var sb strings.Builder
	for {
		i := strings.IndexByte(s, '\x1b')
		if i < 0 {
			sb.WriteString(s)
			return sb.String()
		}
		sb.WriteString(s[:i])
		j := strings.IndexByte(s[i:], 'm')
		if j < 0 {
			return sb.String()
		}
		s = s[i+j+1:]
	}
}
