package table

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
	DataBytes int64     `json:"dataBytes"`
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
				DataBytes: 4096,
				SampledAt: time.Date(2026, 10, 12, 9, 31, 0, 0, time.UTC),
				Pending:   3,
			},
		},
		{
			Name: "a-much-longer-queue-name",
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
				"NAME                      CREATED_AT            STATS_VISIBLE  STATS_DATA_BYTES  STATS_SAMPLED_AT      STATS_PENDING",
				"orders                    2026-10-12T09:30:00Z  12             4096              2026-10-12T09:31:00Z  3",
				"a-much-longer-queue-name  0001-01-01T00:00:00Z  0              0                 -                     -",
			),
		},
		{
			name:    "selected columns in order",
			columns: []string{"stats.visible", "name"},
			rows:    rows,
			exp: lines(
				"STATS_VISIBLE  NAME",
				"12             orders",
				"0              a-much-longer-queue-name",
			),
		},
		{
			name:    "omitted zero prints a dash",
			columns: []string{"name", "stats.sampledAt", "stats.pending", "stats.visible"},
			rows:    rows,
			exp: lines(
				"NAME                      STATS_SAMPLED_AT      STATS_PENDING  STATS_VISIBLE",
				"orders                    2026-10-12T09:31:00Z  3              12",
				"a-much-longer-queue-name  -                     -              0",
			),
		},
		{
			name:    "no rows",
			columns: []string{"name"},
			exp: lines(
				"NAME",
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
					"table:\n%s\nwant:\n%s",
					act, test.exp,
				)
			}
		})
	}
}

func TestHeader(t *testing.T) {
	for _, test := range []struct {
		name string
		in   string
		exp  string
	}{
		{
			name: "empty",
			in:   "",
			exp:  "",
		},
		{
			name: "one word",
			in:   "name",
			exp:  "NAME",
		},
		{
			name: "two words",
			in:   "createdAt",
			exp:  "CREATED_AT",
		},
		{
			name: "three words",
			in:   "leaseTimeoutSeconds",
			exp:  "LEASE_TIMEOUT_SECONDS",
		},
		{
			name: "already upper",
			in:   "UID",
			exp:  "UID",
		},
		{
			name: "acronym at the end",
			in:   "userID",
			exp:  "USER_ID",
		},
		{
			name: "acronym in the middle",
			in:   "maxHTTPRetries",
			exp:  "MAX_HTTP_RETRIES",
		},
		{
			name: "acronym at the start",
			in:   "httpStatus",
			exp:  "HTTP_STATUS",
		},
		{
			name: "digits end a word",
			in:   "sha256Sum",
			exp:  "SHA256_SUM",
		},
		{
			name: "digits inside a word",
			in:   "ipv4Address",
			exp:  "IPV4_ADDRESS",
		},
		{
			name: "nested path",
			in:   "stats.dataBytes",
			exp:  "STATS_DATA_BYTES",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			act := header(test.in)
			if act != test.exp {
				t.Errorf(
					"header(%q): %q; want %q",
					test.in, act, test.exp,
				)
			}
		})
	}
}

func lines(s ...string) string {
	return strings.Join(s, "\n") + "\n"
}
