package main

import (
	"flag"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-cli/internal/flagvars"
	"github.com/spoolersh/spooler-cli/internal/format"
	"github.com/spoolersh/spooler-cli/internal/format/csv"
	"github.com/spoolersh/spooler-cli/internal/format/json"
	"github.com/spoolersh/spooler-cli/internal/format/table"
	"github.com/spoolersh/spooler-cli/internal/format/template"
	"github.com/spoolersh/spooler-cli/internal/format/text"
)

type output[T any] struct {
	columns      *format.Columns[T]
	columnsGiven bool

	csv   csv.Encoding[T]
	json  json.Encoding[T]
	table table.Encoding[T]
	text  text.Encoding[T]

	// formats are the names --output accepts; nil means every format.
	formats     []string
	name        string
	enc         format.Encoding[T]
	outputGiven bool

	// tmpl is --template, which replaces --output when given.
	tmpl *template.Encoding[T]

	noColor bool
	human   bool
	compact bool
}

// makeOutput returns an output defaulting to the format def and the columns
// c, accepting the given formats, or every format when none is given.
func makeOutput[T any](def string, c *format.Columns[T], formats ...string) output[T] {
	return output[T]{
		columns: c,
		formats: formats,
		name:    def,
	}
}

func (t *output[T]) defineFlags(fs *flag.FlagSet) {
	fs.Var(
		flagvars.Opt(format.ColumnsFlagVar(&t.columns), &t.columnsGiven),
		"columns",
		fmt.Sprintf(
			"comma-separated list of columns to print, as `glob` patterns"+
				"; available columns: %s",
			strings.Join(format.PathsFor[T](), ", "),
		),
	)

	encodings := map[string]format.Encoding[T]{
		"csv":   &t.csv,
		"json":  &t.json,
		"table": &t.table,
		"text":  &t.text,
	}
	if t.formats != nil {
		maps.DeleteFunc(encodings, func(name string, _ format.Encoding[T]) bool {
			return !slices.Contains(t.formats, name)
		})
	}
	names := slices.Sorted(maps.Keys(encodings))
	t.enc = encodings[t.name]
	if t.enc == nil {
		panic(fmt.Sprintf("output: default format %q is not available", t.name))
	}
	fs.Var(
		flagvars.Opt(&flagvars.FuncVar{
			DoSet: func(s string) error {
				enc, has := encodings[s]
				if !has {
					return fmt.Errorf(
						"unknown output format %q; available values: %s",
						s, strings.Join(names, ", "),
					)
				}
				t.name, t.enc = s, enc
				return nil
			},
			DoString: func() string {
				return t.name
			},
		}, &t.outputGiven),
		"output",
		fmt.Sprintf(
			"output format; available values: %s",
			strings.Join(names, ", "),
		),
	)

	fs.Var(
		&flagvars.FuncVar{
			DoSet: func(s string) error {
				tmpl, err := template.Parse[T](s)
				if err != nil {
					return err
				}
				t.tmpl = tmpl
				return nil
			},
			DoString: func() string {
				if t.tmpl == nil {
					return ""
				}
				return t.tmpl.Text()
			},
		},
		"template",
		"Go text/template rendering each result's fields by their -o json"+
			" names ({{.id}}, {{.stats.visible}}) as one line; excludes"+
			" --output and --columns",
	)

	fs.BoolVar(&t.noColor,
		"no-color", false,
		"no colour on a terminal; NO_COLOR in the environment does the same",
	)
	fs.BoolVar(&t.human,
		"human", false,
		"with -o table or -o text, print times as ages (3h), seconds as"+
			" durations (7d), byte counts in binary units (1.5MiB) and rate"+
			" limits as count/period (100/s)",
	)
	fs.BoolVar(&t.compact,
		"compact", false,
		"with -o json, print each document on one line instead of indented",
	)
}

// addDefaultColumns appends the columns at names to the default selection,
// for fields another flag made worth showing; it does nothing when --columns
// chose the columns.
func (t *output[T]) addDefaultColumns(names ...string) error {
	if t.columnsGiven || len(names) == 0 {
		return nil
	}
	cols, err := format.ColumnsFor[T](slices.Concat(t.columns.Paths(), names)...)
	if err != nil {
		return err
	}
	t.columns = cols
	return nil
}

// removeDefaultColumns drops the columns at names from the default selection,
// for fields the other flags leave empty; it does nothing when --columns chose
// the columns. The help still lists the columns, since the help is the
// default's.
func (t *output[T]) removeDefaultColumns(names ...string) error {
	if t.columnsGiven || len(names) == 0 {
		return nil
	}
	paths := t.columns.Paths()
	for _, name := range names {
		if !slices.Contains(paths, name) {
			return fmt.Errorf(
				"no default column %q; default columns: %s",
				name, strings.Join(paths, ", "),
			)
		}
	}
	rest := slices.DeleteFunc(slices.Clone(paths), func(p string) bool {
		return slices.Contains(names, p)
	})
	cols, err := format.ColumnsFor[T](rest...)
	if err != nil {
		return err
	}
	t.columns = cols
	return nil
}

// unusedError is the usage error for a rendering flag given when nothing
// renders through this output; with is the flag that would make it count,
// such as "--meta-file -". It is nil when no rendering flag was given.
func (t *output[T]) unusedError(with string) error {
	var flag string
	switch {
	case t.tmpl != nil:
		flag = "--template"
	case t.outputGiven:
		flag = "--output"
	case t.columnsGiven:
		flag = "--columns"
	case t.human:
		flag = "--human"
	case t.compact:
		flag = "--compact"
	default:
		return nil
	}
	return cli.Exitf(2,
		"%s takes effect only with %s",
		flag, with,
	)
}

// encoderTo returns the encoder the flags selected, writing to w, with the
// text keys coloured when w is a terminal that wants colour and the json
// documents indented when w is a terminal; a usage error when the flags
// contradict each other.
func (t *output[T]) encoderTo(w io.Writer) (format.Encoder[T], error) {
	tty := isTerminal(w)
	t.text.KeyStyle = ""
	t.text.PathStyle = ""
	t.json.KeyStyle = ""
	t.json.StringStyle = ""
	if tty && !t.noColor && colorEnabled() {
		t.text.KeyStyle = keyStyle
		t.text.PathStyle = pathStyle
		t.json.KeyStyle = keyStyle
		t.json.StringStyle = stringStyle
	}
	// Every document is indented, as jq prints them, whatever w is; only
	// the colour follows the terminal. --compact is jq's -c: one line each.
	t.json.Indent = "  "
	if t.compact {
		t.json.Indent = ""
	}
	if t.tmpl != nil {
		if t.outputGiven {
			return nil, cli.Exitf(2,
				"--template and --output are mutually exclusive",
			)
		}
		if t.columnsGiven {
			return nil, cli.Exitf(2,
				"--template and --columns are mutually exclusive",
			)
		}
		if t.human {
			return nil, cli.Exitf(2,
				"--template and --human are mutually exclusive",
			)
		}
		if t.compact {
			return nil, cli.Exitf(2,
				"--template and --compact are mutually exclusive",
			)
		}
		return t.tmpl.EncoderTo(w), nil
	}
	t.table.Human = t.human
	t.text.Human = t.human
	if t.human && t.name != "table" && t.name != "text" {
		return nil, cli.Exitf(2,
			"--human takes effect only with `-o table` or `-o text`",
		)
	}
	if t.compact && t.name != "json" {
		return nil, cli.Exitf(2,
			"--compact takes effect only with `-o json`",
		)
	}
	err := format.SetColumns(t.enc, t.columns)
	if err != nil && t.columnsGiven {
		return nil, cli.Exitf(2,
			"--columns: %v",
			err,
		)
	}
	return t.enc.EncoderTo(w), nil
}
