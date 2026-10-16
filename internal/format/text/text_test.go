package text

import (
	"strings"
	"testing"
	"time"

	"github.com/spoolersh/spooler-cli/internal/format"
)

type testRow struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Stats     testStats `json:"stats"`
}

type testStats struct {
	Visible   int64     `json:"visible"`
	SampledAt time.Time `json:"sampledAt,omitzero"`
	Pending   int64     `json:"pending,omitempty"`
}

func TestEncoder(t *testing.T) {
	rows := []testRow{
		{
			Name:      "orders",
			CreatedAt: time.Date(2026, 10, 12, 9, 30, 0, 0, time.UTC),
			Stats: testStats{
				Visible:   12,
				SampledAt: time.Date(2026, 10, 12, 9, 31, 0, 0, time.UTC),
				Pending:   3,
			},
		},
		{
			Name: "refunds",
		},
	}
	for _, test := range []struct {
		name      string
		columns   []string
		style     string
		pathStyle string
		rows      []testRow
		exp       string
	}{
		{
			name:    "styled keys",
			columns: []string{"name", "stats.visible"},
			style:   "\x1b[38;5;227m",
			rows:    rows[:1],
			exp: lines(
				"\x1b[38;5;227mname\x1b[0m: orders",
				"\x1b[38;5;227mstats.visible\x1b[0m: 12",
				"",
			),
		},
		{
			name:      "styled path and leaf",
			columns:   []string{"name", "stats.visible"},
			style:     "\x1b[38;5;227m",
			pathStyle: "\x1b[2m",
			rows:      rows[:1],
			exp: lines(
				"\x1b[38;5;227mname\x1b[0m: orders",
				"\x1b[2mstats.\x1b[0m\x1b[38;5;227mvisible\x1b[0m: 12",
				"",
			),
		},
		{
			name:      "path style alone styles nothing",
			columns:   []string{"stats.visible"},
			pathStyle: "\x1b[2m",
			rows:      rows[:1],
			exp: lines(
				"stats.visible: 12",
				"",
			),
		},
		{
			name: "every leaf",
			rows: rows,
			exp: lines(
				"name: orders",
				"createdAt: 2026-10-12T09:30:00Z",
				"stats.visible: 12",
				"stats.sampledAt: 2026-10-12T09:31:00Z",
				"stats.pending: 3",
				"",
				"name: refunds",
				"createdAt: 0001-01-01T00:00:00Z",
				"stats.visible: 0",
				"",
			),
		},
		{
			name:    "selected columns in order",
			columns: []string{"stats.visible", "name"},
			rows:    rows,
			exp: lines(
				"stats.visible: 12",
				"name: orders",
				"",
				"stats.visible: 0",
				"name: refunds",
				"",
			),
		},
		{
			name:    "a record of omitted fields only is an empty line",
			columns: []string{"stats.sampledAt", "stats.pending"},
			rows:    rows[1:],
			exp: lines(
				"",
			),
		},
		{
			name:    "no rows",
			columns: []string{"name"},
			exp:     "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var sb strings.Builder
			e := Encoding[testRow]{
				KeyStyle:  test.style,
				PathStyle: test.pathStyle,
			}
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
					"text:\n%s\nwant:\n%s",
					act, test.exp,
				)
			}
		})
	}
}

// lines joins each line with a newline, the last one included.
func lines(ls ...string) string {
	return strings.Join(ls, "\n") + "\n"
}
