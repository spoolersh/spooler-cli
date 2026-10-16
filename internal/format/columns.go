package format

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/gobwas/glob"
	"github.com/spoolersh/spooler-cli/internal/flagvars"
	"github.com/spoolersh/spooler-cli/internal/format/internal/fields"
)

// Columns is an ordered selection of T's leaf fields, from ColumnsFor, for
// the formats that lay a value out flat.
//
// Nil selects every leaf of T in field order.
type Columns[T any] struct {
	paths []string
	index []int
}

// ColumnsFor resolves json paths, such as "name" or "stats.visible", to the
// leaf fields of T they name, in the given order. A path is a glob pattern
// (github.com/gobwas/glob): "stats.*" selects every field under stats in
// field order, "*Total" every field ending so, "stats.{visible,pending}"
// both, "*" every field. A field selected twice is kept at its first
// position.
//
// It returns an error for no paths, a malformed pattern, or a pattern
// matching no leaf field of T.
func ColumnsFor[T any](paths ...string) (*Columns[T], error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf(
			"no columns selected",
		)
	}
	var (
		fs    = fields.Of(*new(T))
		index = make([]int, 0, len(paths))
		seen  = make(map[int]bool, len(paths))
	)
	for _, path := range paths {
		g, err := glob.Compile(path)
		if err != nil {
			return nil, fmt.Errorf(
				"column %q: %w",
				path, err,
			)
		}
		n := 0
		for i, f := range fs {
			if !g.Match(f.Path) {
				continue
			}
			n++
			if !seen[i] {
				seen[i] = true
				index = append(index, i)
			}
		}
		if n == 0 {
			return nil, fmt.Errorf(
				"no column matches %q; available columns: %s",
				path, strings.Join(fields.Paths(fs), ", "),
			)
		}
	}
	return &Columns[T]{
		paths: paths,
		index: index,
	}, nil
}

// ColumnsFlagVar returns a flag value parsing a comma-separated list of
// patterns, as ColumnsFor takes them, into *p; *p is the default it prints.
func ColumnsFlagVar[T any](p **Columns[T]) flag.Value {
	return &flagvars.FuncVar{
		DoSet: func(s string) error {
			cols, err := ColumnsFor[T](splitColumns(s)...)
			if err != nil {
				return err
			}
			*p = cols
			return nil
		},
		DoString: func() string {
			return (*p).String()
		},
	}
}

// PathsFor returns the json paths of every leaf field of T, in field order.
func PathsFor[T any](...T) []string {
	return fields.Paths(fields.Of(*new(T)))
}

// splitColumns splits a --columns value on the commas outside braces and trims
// each part, so a pattern's alternatives ("stats.{visible,pending}") stay
// together.
func splitColumns(s string) []string {
	var (
		ret   []string
		depth int
		start int
	)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++ // The next character is literal, whatever it is.
		case '{':
			depth++
		case '}':
			depth = max(depth-1, 0)
		case ',':
			if depth == 0 {
				ret = append(ret, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	return append(ret, strings.TrimSpace(s[start:]))
}

// String returns the selected paths comma-separated, as --columns takes
// them; empty for nil.
func (c *Columns[T]) String() string {
	if c == nil {
		return ""
	}
	return strings.Join(c.paths, ",")
}

// Paths returns the selected columns' json paths, in order; every leaf's
// for nil.
func (c *Columns[T]) Paths() []string {
	fs := fields.Of(*new(T))
	index := c.indices(len(fs))
	ret := make([]string, len(index))
	for i, j := range index {
		ret[i] = fs[j].Path
	}
	return ret
}

// Cell is one value of a row.
type Cell struct {
	// Text is the value's text; empty when Omitted.
	Text string

	// Omitted reports a value the json format leaves out, for a format to
	// mark its own way.
	Omitted bool
}

// Row renders v's selected columns, in order. With human, a field tagged
// `human` renders in the form the tag names (an age, a duration, a rate, a
// byte count) instead of its json text.
func (c *Columns[T]) Row(v T, human bool) ([]Cell, error) {
	fs := fields.Of(v)
	index := c.indices(len(fs))
	ret := make([]Cell, len(index))
	now := time.Now()
	for i, j := range index {
		f := fs[j]
		if fields.Omitted(f) {
			ret[i] = Cell{
				Omitted: true,
			}
			continue
		}
		var (
			s   string
			ok  bool
			err error
		)
		if human {
			s, ok, err = fields.Human(f, now)
		}
		if err == nil && !ok {
			s, err = fields.Text(f)
		}
		if err != nil {
			return nil, fmt.Errorf(
				"format %q: %w",
				f.Path, err,
			)
		}
		ret[i] = Cell{
			Text: s,
		}
	}
	return ret, nil
}

// indices returns the selected fields' indices, or every index below n for
// nil Columns.
func (c *Columns[T]) indices(n int) []int {
	if c != nil {
		return c.index
	}
	ret := make([]int, n)
	for i := range ret {
		ret[i] = i
	}
	return ret
}
