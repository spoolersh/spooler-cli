package csv

import (
	"strings"
	"testing"
	"time"

	"github.com/spoolersh/spooler-cli/internal/format"
)

type testRow struct {
	Name  string    `json:"name"`
	Stats testStats `json:"stats"`
}

type testStats struct {
	Visible   int64     `json:"visible"`
	SampledAt time.Time `json:"sampledAt,omitzero"`
	Pending   int64     `json:"pending,omitempty"`
}

func TestEncoder(t *testing.T) {
	rows := []testRow{
		{
			Name: "orders, today",
			Stats: testStats{
				Visible:   12,
				SampledAt: time.Date(2026, 10, 12, 9, 31, 0, 0, time.UTC),
				Pending:   3,
			},
		},
		{
			Name: "plain",
		},
	}
	for _, test := range []struct {
		name    string
		columns []string
		rows    []testRow
		exp     string
	}{
		{
			name: "every leaf",
			rows: rows,
			exp: lines(
				"name,stats.visible,stats.sampledAt,stats.pending",
				`"orders, today",12,2026-10-12T09:31:00Z,3`,
				"plain,0,,",
			),
		},
		{
			name:    "selected columns in order",
			columns: []string{"stats.pending", "name"},
			rows:    rows,
			exp: lines(
				"stats.pending,name",
				`3,"orders, today"`,
				",plain",
			),
		},
		{
			name:    "no rows",
			columns: []string{"name"},
			exp: lines(
				"name",
			),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var sb strings.Builder
			var e Encoding[testRow]
			if test.columns != nil {
				cols, err := format.ColumnsFor[testRow](test.columns...)
				if err != nil {
					t.Fatalf("ColumnsFor: %v", err)
				}
				e.SetColumns(cols)
			}
			enc := e.EncoderTo(&sb)
			for _, row := range test.rows {
				if err := enc.Encode(row); err != nil {
					t.Fatalf("Encode: %v", err)
				}
			}
			if err := enc.Flush(); err != nil {
				t.Fatalf("Flush: %v", err)
			}
			if act := sb.String(); act != test.exp {
				t.Errorf(
					"csv:\n%s\nwant:\n%s",
					act, test.exp,
				)
			}
		})
	}
}

func lines(s ...string) string {
	return strings.Join(s, "\n") + "\n"
}
