package flagvars

import (
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

func Bypass(s string) (string, error) {
	return s, nil
}

func Validate(v flag.Value, fns ...func(s string) error) flag.Value {
	return &FuncVar{
		DoSet: func(s string) error {
			for _, f := range fns {
				if err := f(s); err != nil {
					return err
				}
			}
			return v.Set(s)
		},
		DoString: func() string {
			return v.String()
		},
		DoIsBoolFlag: func() bool {
			return isBoolFlag(v)
		},
	}
}

func FilePathVar(p *string) flag.Value {
	return &FuncVar{
		DoSet: func(name string) error {
			info, err := os.Stat(name)
			if err != nil {
				return err
			}
			if info.IsDir() {
				return fmt.Errorf(
					"%s is a directory",
					name,
				)
			}
			*p = name
			return nil
		},
		DoString: func() string {
			if p == nil {
				return ""
			}
			return *p
		},
	}
}

func ReadFileVar(p *string) flag.Value {
	var name string
	return &FuncVar{
		DoSet: func(s string) error {
			file, err := os.Open(s)
			if err != nil {
				return err
			}
			defer file.Close()
			var sb strings.Builder
			if _, err := io.Copy(&sb, file); err != nil {
				return err
			}
			name = s
			*p = sb.String()
			return nil
		},
		DoString: func() string {
			return fmt.Sprintf(
				"<file at %s>",
				name,
			)
		},
	}
}

// Opt wraps v so that *given reports whether the flag was set, for a flag
// whose omission means something a default value cannot. The help shows v's
// default as it would without the wrapper.
func Opt(v flag.Value, given *bool) flag.Value {
	return &FuncVar{
		DoSet: func(s string) error {
			if err := v.Set(s); err != nil {
				return err
			}
			*given = true
			return nil
		},
		DoString: v.String,
		DoIsBoolFlag: func() bool {
			return isBoolFlag(v)
		},
	}
}

func OptVar[T any](p **T, conv func(string) (T, error)) flag.Value {
	return &FuncVar{
		DoSet: func(s string) error {
			v, err := conv(s)
			if err != nil {
				return err
			}
			*p = &v
			return nil
		},
		DoString: func() string {
			if p == nil || *p == nil {
				return ""
			}
			return fmt.Sprintf("%v", **p)
		},
		DoIsBoolFlag: isBool[T],
	}
}

func Ident[T ~string](s string) (T, error) {
	return T(s), nil
}

func StringVar[T ~string](p *T) flag.Value {
	return ConvFuncVar(p, Ident[T])
}
func IntVar[T ~int](p *T) flag.Value {
	return ConvFuncVar(p, func(s string) (zero T, err error) {
		i, err := strconv.ParseInt(s, 10, strconv.IntSize)
		if err != nil {
			return zero, err
		}
		return T(i), nil
	})
}

func ConvFuncVar[T any](p *T, conv func(string) (T, error)) flag.Value {
	return &resetOnceVar{
		value: &convFuncVar[T]{
			conv: conv,
			dest: p,
		},
	}
}

type convFuncVar[T any] struct {
	conv func(string) (T, error)
	dest *T
}

func (f *convFuncVar[T]) reset() {
	var zero T
	*f.dest = zero
}

func (f *convFuncVar[T]) Set(s string) error {
	v, err := f.conv(s)
	if err != nil {
		return err
	}
	*f.dest = v
	return nil
}

// String renders the value, or nothing for the zero value of T, so help
// prints no default for a flag that has none.
func (f *convFuncVar[T]) String() string {
	if f == nil || f.dest == nil || reflect.ValueOf(*f.dest).IsZero() {
		return ""
	}
	return fmt.Sprintf("%v", *f.dest)
}

func (f *convFuncVar[T]) IsBoolFlag() bool {
	return isBool[T]()
}

func SliceVar[T any](p *[]T, conv func(string) (T, error)) flag.Value {
	return &resetOnceVar{
		value: &sliceVar[T]{
			delim: ",",
			conv:  conv,
			dest:  p,
		},
	}
}

type sliceVar[T any] struct {
	delim string
	conv  func(string) (T, error)
	dest  *[]T
}

func (s *sliceVar[T]) reset() {
	var zero []T
	*s.dest = zero
}

func (s *sliceVar[T]) String() string {
	if s == nil || s.dest == nil {
		return ""
	}
	var sb strings.Builder
	for i, v := range *s.dest {
		if i > 0 {
			sb.WriteString(s.delim)
		}
		fmt.Fprintf(&sb, "%v", v)
	}
	return sb.String()
}

func (s *sliceVar[T]) Set(str string) error {
	for len(str) > 0 {
		var val string
		val, str, _ = strings.Cut(str, s.delim)
		val = strings.TrimSpace(val)
		if len(val) == 0 {
			continue
		}
		v, err := s.conv(val)
		if err != nil {
			return err
		}
		*s.dest = append(*s.dest, v)
	}
	return nil
}

func MapVar[K comparable, V any, M ~map[K]V](p *M, convKey func(string) (K, error), convVal func(string) (V, error)) flag.Value {
	return &resetOnceVar{
		value: &mapVar[K, V, M]{
			delim:   ",",
			assign:  ":",
			convKey: convKey,
			convVal: convVal,
			dest:    p,
		},
	}
}

type mapVar[K comparable, V any, M ~map[K]V] struct {
	delim   string
	assign  string
	convKey func(string) (K, error)
	convVal func(string) (V, error)
	dest    *M
}

func (m *mapVar[K, V, M]) reset() {
	*m.dest = make(M)
}

func (m *mapVar[K, V, M]) String() string {
	if m == nil || m.dest == nil {
		return ""
	}
	var sb strings.Builder
	for k, v := range *m.dest {
		if sb.Len() > 0 {
			sb.WriteString(m.delim)
		}
		fmt.Fprintf(&sb, "%v", k)
		sb.WriteString(m.assign)
		fmt.Fprintf(&sb, "%v", v)
	}
	return sb.String()
}

func (m *mapVar[K, V, M]) Set(str string) error {
	for len(str) > 0 {
		var pair string
		pair, str, _ = strings.Cut(str, m.delim)
		pair = strings.TrimSpace(pair)
		if len(pair) == 0 {
			continue
		}
		key, val, _ := strings.Cut(pair, m.assign)
		k, err := m.convKey(strings.TrimSpace(key))
		if err != nil {
			return err
		}
		v, err := m.convVal(strings.TrimSpace(val))
		if err != nil {
			return err
		}
		(*m.dest)[k] = v
	}
	return nil
}

type resetValue interface {
	flag.Value
	reset()
}

type resetOnceVar struct {
	value resetValue
	once  sync.Once
}

func (r *resetOnceVar) String() string {
	if r == nil || r.value == nil {
		return ""
	}
	return r.value.String()
}

func (r *resetOnceVar) Set(s string) error {
	r.once.Do(func() {
		r.value.reset()
	})
	return r.value.Set(s)
}

func (r *resetOnceVar) IsBoolFlag() bool {
	return r != nil && isBoolFlag(r.value)
}

type FuncVar struct {
	DoSet    func(string) error
	DoString func() string

	// DoIsBoolFlag reports a boolean flag, one a parser accepts without a
	// value; nil means not one.
	DoIsBoolFlag func() bool
}

func (w *FuncVar) Set(s string) error {
	return w.DoSet(s)
}

// IsBoolFlag is what the flag package and pargs ask to tell a boolean flag.
func (w *FuncVar) IsBoolFlag() bool {
	return w != nil && w.DoIsBoolFlag != nil && w.DoIsBoolFlag()
}

// isBool reports whether T is bool.
func isBool[T any]() bool {
	_, ok := any(new(T)).(*bool)
	return ok
}

// isBoolFlag reports whether v says it is a boolean flag.
func isBoolFlag(v flag.Value) bool {
	b, ok := v.(interface {
		IsBoolFlag() bool
	})
	return ok && b.IsBoolFlag()
}

func (w *FuncVar) String() string {
	if w == nil {
		return "<nil>"
	}
	if w.DoString == nil {
		return ""
	}
	return w.DoString()
}
