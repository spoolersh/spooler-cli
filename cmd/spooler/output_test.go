package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spoolersh/spooler-cli/internal/format"
)

func TestAddDefaultColumns(t *testing.T) {
	for _, test := range []struct {
		name    string
		columns []string
		given   bool
		names   []string
		exp     []string
		err     bool
	}{
		{
			name:    "appended to the default",
			columns: []string{"id", "state"},
			names:   []string{"handle", "visibleAt"},
			exp:     []string{"id", "state", "handle", "visibleAt"},
		},
		{
			name:    "already selected keeps its position",
			columns: []string{"id", "handle", "state"},
			names:   []string{"handle"},
			exp:     []string{"id", "handle", "state"},
		},
		{
			name:  "every column stays every column",
			names: []string{"handle"},
			exp:   format.PathsFor[messageOutput](),
		},
		{
			name:    "none keeps the default",
			columns: []string{"id", "state"},
			exp:     []string{"id", "state"},
		},
		{
			name:    "columns given are kept",
			columns: []string{"id", "state"},
			given:   true,
			names:   []string{"handle"},
			exp:     []string{"id", "state"},
		},
		{
			name:    "unknown name is an error",
			columns: []string{"id"},
			names:   []string{"handle", "nope"},
			err:     true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var cols *format.Columns[messageOutput]
			if test.columns != nil {
				cols = must(format.ColumnsFor[messageOutput](test.columns...))
			}
			o := makeOutput("table", cols)
			o.columnsGiven = test.given
			err := o.addDefaultColumns(test.names...)
			if (err != nil) != test.err {
				t.Fatalf("error: %v; want error %t", err, test.err)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(test.exp, o.columns.Paths()); diff != "" {
				t.Errorf("columns (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRemoveDefaultColumns(t *testing.T) {
	for _, test := range []struct {
		name    string
		columns []string
		given   bool
		names   []string
		exp     []string
		err     bool
	}{
		{
			name:    "removed from the default",
			columns: []string{"id", "handle", "state", "visibleAt"},
			names:   []string{"handle", "visibleAt"},
			exp:     []string{"id", "state"},
		},
		{
			name:  "removed from every column",
			names: []string{"handle"},
			exp:   []string{"id", "state", "retries", "createdAt", "visibleAt", "leaseExpiresAt", "expiresAt"},
		},
		{
			name:    "none keeps the default",
			columns: []string{"id", "handle"},
			exp:     []string{"id", "handle"},
		},
		{
			name:    "columns given are kept",
			columns: []string{"id", "handle"},
			given:   true,
			names:   []string{"handle"},
			exp:     []string{"id", "handle"},
		},
		{
			name:    "not a default column is an error",
			columns: []string{"id", "state"},
			names:   []string{"id", "handle"},
			err:     true,
		},
		{
			name:    "the last column is an error",
			columns: []string{"id", "handle"},
			names:   []string{"id", "handle"},
			err:     true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var cols *format.Columns[messageOutput]
			if test.columns != nil {
				cols = must(format.ColumnsFor[messageOutput](test.columns...))
			}
			o := makeOutput("table", cols)
			o.columnsGiven = test.given
			err := o.removeDefaultColumns(test.names...)
			if (err != nil) != test.err {
				t.Fatalf("error: %v; want error %t", err, test.err)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(test.exp, o.columns.Paths()); diff != "" {
				t.Errorf("columns (-want +got):\n%s", diff)
			}
		})
	}
}
