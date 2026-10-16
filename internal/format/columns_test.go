package format

import (
	"flag"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/spoolersh/spooler-cli/internal/flagvars"
)

type testRow struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Stats     testStats `json:"stats"`
}

type testStats struct {
	Visible   int64     `json:"visible"`
	SampledAt time.Time `json:"sampledAt,omitzero"`
}

func TestColumns(t *testing.T) {
	row := testRow{
		Name:      "orders",
		CreatedAt: time.Date(2026, 10, 12, 9, 30, 0, 0, time.UTC),
		Stats: testStats{
			Visible: 12,
		},
	}
	for _, test := range []struct {
		name    string
		columns []string
		paths   []string
		row     []Cell
	}{
		{
			name:    "every leaf",
			columns: nil,
			paths:   []string{"name", "createdAt", "stats.visible", "stats.sampledAt"},
			row: []Cell{
				{Text: "orders"},
				{Text: "2026-10-12T09:30:00Z"},
				{Text: "12"},
				{Omitted: true},
			},
		},
		{
			name:    "selected in order",
			columns: []string{"stats.visible", "name"},
			paths:   []string{"stats.visible", "name"},
			row: []Cell{
				{Text: "12"},
				{Text: "orders"},
			},
		},
		{
			name:    "star selects a subtree in field order",
			columns: []string{"stats.*", "name"},
			paths:   []string{"stats.visible", "stats.sampledAt", "name"},
			row: []Cell{
				{Text: "12"},
				{Omitted: true},
				{Text: "orders"},
			},
		},
		{
			name:    "star alone selects every leaf",
			columns: []string{"*"},
			paths:   []string{"name", "createdAt", "stats.visible", "stats.sampledAt"},
			row: []Cell{
				{Text: "orders"},
				{Text: "2026-10-12T09:30:00Z"},
				{Text: "12"},
				{Omitted: true},
			},
		},
		{
			name:    "star in the middle",
			columns: []string{"*.*At"},
			paths:   []string{"stats.sampledAt"},
			row: []Cell{
				{Omitted: true},
			},
		},
		{
			name:    "braces select alternatives in field order",
			columns: []string{"stats.{sampledAt,visible}"},
			paths:   []string{"stats.visible", "stats.sampledAt"},
			row: []Cell{
				{Text: "12"},
				{Omitted: true},
			},
		},
		{
			name:    "a leaf selected twice keeps its first position",
			columns: []string{"stats.visible", "*"},
			paths:   []string{"stats.visible", "name", "createdAt", "stats.sampledAt"},
			row: []Cell{
				{Text: "12"},
				{Text: "orders"},
				{Text: "2026-10-12T09:30:00Z"},
				{Omitted: true},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var cols *Columns[testRow]
			if test.columns != nil {
				var err error
				cols, err = ColumnsFor[testRow](test.columns...)
				if err != nil {
					t.Fatalf("ColumnsFor: %v", err)
				}
			}
			if diff := cmp.Diff(test.paths, cols.Paths()); diff != "" {
				t.Errorf("paths:\n%s", diff)
			}
			act, err := cols.Row(row, false)
			if err != nil {
				t.Fatalf("Row: %v", err)
			}
			if diff := cmp.Diff(test.row, act); diff != "" {
				t.Errorf("row:\n%s", diff)
			}
		})
	}
}

func TestColumnsForError(t *testing.T) {
	for _, test := range []struct {
		name  string
		paths []string
	}{
		{
			name:  "no paths",
			paths: nil,
		},
		{
			name:  "unknown path",
			paths: []string{"name", "nope"},
		},
		{
			name:  "struct path",
			paths: []string{"stats"},
		},
		{
			name:  "header instead of path",
			paths: []string{"STATS_VISIBLE"},
		},
		{
			name:  "pattern matching nothing",
			paths: []string{"settings.*"},
		},
		{
			name:  "malformed pattern",
			paths: []string{"stats.{visible"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cols, err := ColumnsFor[testRow](test.paths...)
			if err == nil {
				t.Errorf(
					"ColumnsFor(%q): %v, nil; want an error",
					test.paths, cols,
				)
			}
		})
	}
}

func TestSplitColumns(t *testing.T) {
	for _, test := range []struct {
		in  string
		exp []string
	}{
		{"", []string{""}},
		{"name", []string{"name"}},
		{"name,state", []string{"name", "state"}},
		{"name, state ,uid", []string{"name", "state", "uid"}},
		{"stats.{visible,pending},name", []string{"stats.{visible,pending}", "name"}},
		{"{a,{b,c}},d", []string{"{a,{b,c}}", "d"}},
		{`a\,b,c`, []string{`a\,b`, "c"}},
		{"a},b", []string{"a}", "b"}},
		{"a{b,c", []string{"a{b,c"}},
	} {
		t.Run(test.in, func(t *testing.T) {
			if diff := cmp.Diff(test.exp, splitColumns(test.in)); diff != "" {
				t.Errorf("splitColumns(%q):\n%s", test.in, diff)
			}
		})
	}
}

func TestColumnsFlagVar(t *testing.T) {
	for _, test := range []struct {
		name  string
		def   []string
		args  []string
		exp   []string
		given bool
		err   bool
	}{
		{
			name: "not given",
			args: nil,
			exp:  nil,
		},
		{
			name: "not given keeps the default",
			def:  []string{"name"},
			args: nil,
			exp:  []string{"name"},
		},
		{
			name:  "given",
			args:  []string{"-columns", "stats.visible, name"},
			exp:   []string{"stats.visible", "name"},
			given: true,
		},
		{
			name:  "given replaces the default",
			def:   []string{"name"},
			args:  []string{"-columns", "stats.visible"},
			exp:   []string{"stats.visible"},
			given: true,
		},
		{
			name:  "braces keep their commas",
			args:  []string{"-columns", "stats.{sampledAt,visible}, name"},
			exp:   []string{"stats.{sampledAt,visible}", "name"},
			given: true,
		},
		{
			name: "unknown",
			args: []string{"-columns", "name,nope"},
			err:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var (
				cols  *Columns[testRow]
				given bool
			)
			if test.def != nil {
				cols, _ = ColumnsFor[testRow](test.def...)
			}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			fs.Var(flagvars.Opt(ColumnsFlagVar(&cols), &given), "columns", "")
			if act, exp := fs.Lookup("columns").DefValue, strings.Join(test.def, ","); act != exp {
				t.Errorf("default: %q; want %q", act, exp)
			}
			err := fs.Parse(test.args)
			if (err != nil) != test.err {
				t.Fatalf("Parse(%q): %v; want error: %t", test.args, err, test.err)
			}
			if test.err {
				return
			}
			if given != test.given {
				t.Errorf("given: %t; want %t", given, test.given)
			}
			var exp *Columns[testRow]
			if test.exp != nil {
				exp, _ = ColumnsFor[testRow](test.exp...)
			}
			if diff := cmp.Diff(exp, cols, cmp.AllowUnexported(Columns[testRow]{})); diff != "" {
				t.Errorf("columns:\n%s", diff)
			}
		})
	}
}

func TestPathsFor(t *testing.T) {
	exp := []string{"name", "createdAt", "stats.visible", "stats.sampledAt"}
	if diff := cmp.Diff(exp, PathsFor[testRow]()); diff != "" {
		t.Errorf("PathsFor:\n%s", diff)
	}
}
