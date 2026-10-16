// Package fields walks a value's json-tagged leaf fields, for the formats
// that lay a value out flat.
package fields

import (
	"encoding"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/spoolersh/spooler-cli/internal/human"
)

// Field is one leaf of a value: a field whose value renders as one cell.
type Field struct {
	// Path is the field's json path, as "stats.visible".
	Path string

	// Depth is how many structs the field is nested in.
	Depth int

	// OmitEmpty reports the json tag's omitempty option.
	OmitEmpty bool

	// OmitZero reports the json tag's omitzero option.
	OmitZero bool

	// Human is the field's `human` tag: how Human renders the value, or
	// empty for as is.
	Human string

	// Value is the field's value.
	Value reflect.Value
}

var textMarshaler = reflect.TypeFor[encoding.TextMarshaler]()

// Of walks v's json-tagged fields depth first and returns its leaves in
// field order. The result depends on v's type only, never on its contents,
// so every value of a type yields the same fields in the same order.
func Of[T any](v T) (ret []Field) {
	var walk func(depth int, prefix string, t reflect.Value)
	walk = func(depth int, prefix string, t reflect.Value) {
		for f, v := range t.Fields() {
			tag, has := f.Tag.Lookup("json")
			if tag == "-" || !has {
				continue
			}
			name, rest, _ := strings.Cut(tag, ",")
			if name == "" {
				continue
			}
			path := prefix + name

			if !isLeaf(f.Type) {
				if f.Type.Kind() == reflect.Struct {
					walk(depth+1, path+".", v)
				}
				// Slices, maps and the rest have no place in a cell.
				continue
			}

			field := Field{
				Path:  path,
				Value: v,
				Depth: depth,
				Human: f.Tag.Get("human"),
			}
			for len(rest) > 0 {
				var opt string
				opt, rest, _ = strings.Cut(rest, ",")
				switch opt {
				case "omitempty":
					field.OmitEmpty = true
				case "omitzero":
					field.OmitZero = true
				}
			}
			ret = append(ret, field)
		}
	}
	walk(0, "", reflect.ValueOf(v))
	return
}

// Paths lists the paths of fs, in order.
func Paths(fs []Field) []string {
	ret := make([]string, len(fs))
	for i, f := range fs {
		ret[i] = f.Path
	}
	return ret
}

// isLeaf reports whether t renders as one cell: a text marshaler (such as
// time.Time, which is a struct otherwise), or a scalar.
func isLeaf(t reflect.Type) bool {
	if reflect.PointerTo(t).Implements(textMarshaler) {
		return true
	}
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	}
	return false
}

// Text renders f's value: its text marshaling when it has one, so a time
// prints as the json format does, else as fmt prints it.
func Text(f Field) (string, error) {
	if m, ok := f.Value.Interface().(encoding.TextMarshaler); ok {
		p, err := m.MarshalText()
		if err != nil {
			return "", err
		}
		return string(p), nil
	}
	if f.Value.CanAddr() {
		if m, ok := f.Value.Addr().Interface().(encoding.TextMarshaler); ok {
			p, err := m.MarshalText()
			if err != nil {
				return "", err
			}
			return string(p), nil
		}
	}
	return fmt.Sprint(f.Value.Interface()), nil
}

// Human renders f's value in the form its `human` tag names, relative to
// now, and reports false for a field without the tag:
//
//	age     a time.Time, as the duration between it and now, past or future
//	seconds an integer count of seconds, as a duration
//	rate    an integer interval in microseconds, as a count per period
//	bytes   an integer byte count, in binary units
//
// The forms are those of package human.
func Human(f Field, now time.Time) (string, bool, error) {
	if f.Human == "" {
		return "", false, nil
	}
	v := f.Value
	n, isInt := integer(v)
	switch {
	case f.Human == "age":
		t, ok := v.Interface().(time.Time)
		if !ok {
			break
		}
		return human.FormatDuration(now.Sub(t).Abs()), true, nil
	case !isInt:
	case f.Human == "seconds":
		return human.FormatDuration(time.Duration(n) * time.Second), true, nil
	case f.Human == "rate":
		if n == 0 {
			return "unlimited", true, nil
		}
		return human.FormatRate(time.Duration(n) * time.Microsecond), true, nil
	case f.Human == "bytes":
		if n < 0 {
			break
		}
		return human.FormatBytes(uint64(n)), true, nil
	}
	return "", false, fmt.Errorf(
		"human:%q does not apply to %q of type %s",
		f.Human, f.Path, v.Type(),
	)
}

// integer returns v as an int64 when it is of an integer kind.
func integer(v reflect.Value) (int64, bool) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(v.Uint()), true
	}
	return 0, false
}

// Omitted reports whether f holds a value its json tag leaves out, as
// encoding/json defines it: for omitzero, the zero value or what IsZero
// reports; for omitempty, false, 0, "" or nil, never a struct.
func Omitted(f Field) bool {
	v := f.Value
	if f.OmitZero {
		if z, ok := v.Interface().(interface{ IsZero() bool }); ok {
			return z.IsZero()
		}
		return v.IsZero()
	}
	if f.OmitEmpty && v.Kind() != reflect.Struct {
		return v.IsZero()
	}
	return false
}
