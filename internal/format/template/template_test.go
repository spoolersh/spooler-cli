package template

import (
	"strings"
	"testing"
	"time"
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
}

func TestEncoder(t *testing.T) {
	rows := []testRow{
		{
			Name:      "orders",
			CreatedAt: time.Date(2026, 10, 12, 9, 30, 0, 0, time.UTC),
			Stats: testStats{
				Visible:   12,
				DataBytes: 1200000,
				SampledAt: time.Date(2026, 10, 12, 9, 31, 0, 0, time.UTC),
			},
		},
		{
			Name: "refunds",
		},
	}
	// Two numbers a string comparison would order the other way.
	digits := testRow{
		Name: "digits",
		Stats: testStats{
			Visible:   10,
			DataBytes: 9,
		},
	}
	for _, test := range []struct {
		name     string
		template string
		rows     []testRow
		exp      string
		parseErr bool
		execErr  bool
		// errText is the execution error's message, when one is expected.
		errText string
	}{
		{
			name:     "json names",
			template: "{{.name}} {{.stats.visible}}",
			rows:     rows,
			exp:      "orders 12\nrefunds 0\n",
		},
		{
			name:     "integers stay integers",
			template: "{{.stats.dataBytes}}",
			rows:     rows[:1],
			exp:      "1200000\n",
		},
		{
			name:     "zero is false",
			template: "{{if .stats.visible}}some{{else}}none{{end}}",
			rows:     rows,
			exp:      "some\nnone\n",
		},
		{
			name:     "compares with a literal",
			template: "{{if gt .stats.visible 10}}many{{else}}few{{end}}",
			rows:     rows,
			exp:      "many\nfew\n",
		},
		{
			name:     "compares as numbers",
			template: "{{if lt .stats.visible .stats.dataBytes}}lt{{else}}ge{{end}}",
			rows:     []testRow{digits},
			exp:      "ge\n",
		},
		{
			name:     "time keeps its methods",
			template: "{{.createdAt.Year}} {{.createdAt.Format \"2006-01-02\"}}",
			rows:     rows[:1],
			exp:      "2026 2026-10-12\n",
		},
		{
			name:     "omitted field is false",
			template: "{{if .stats.sampledAt}}{{.stats.sampledAt}}{{else}}-{{end}}",
			rows:     rows,
			exp:      "2026-10-12 09:31:00 +0000 UTC\n-\n",
		},
		{
			name:     "omitted field prints no value",
			template: "{{.stats.sampledAt}}",
			rows:     rows[1:],
			exp:      "<no value>\n",
		},
		{
			name:     "unknown field is an error naming the fields",
			template: "{{.created_af}}",
			rows:     rows[:1],
			execErr:  true,
			errText: `template: --template:1:2: executing "--template" at` +
				` <.created_af>: map has no entry for key "created_af";` +
				` fields: name, createdAt, stats.visible, stats.dataBytes,` +
				` stats.sampledAt`,
			exp: "",
		},
		{
			name:     "unknown nested field is an error naming the fields",
			template: "{{.stats.visibel}}",
			rows:     rows[:1],
			execErr:  true,
			errText: `template: --template:1:8: executing "--template" at` +
				` <.stats.visibel>: map has no entry for key "visibel";` +
				` fields: name, createdAt, stats.visible, stats.dataBytes,` +
				` stats.sampledAt`,
			exp: "",
		},
		{
			name:     "no rows",
			template: "{{.name}}",
			exp:      "",
		},
		{
			name:     "parse error",
			template: "{{.name",
			parseErr: true,
		},
		{
			name:     "execution error writes nothing",
			template: "{{.name.nope}}",
			rows:     rows[:1],
			execErr:  true,
			exp:      "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			e, err := Parse[testRow](test.template)
			if (err != nil) != test.parseErr {
				t.Fatalf("Parse(%q): %v; want error: %t", test.template, err, test.parseErr)
			}
			if test.parseErr {
				return
			}
			if act := e.Text(); act != test.template {
				t.Errorf("Text: %q; want %q", act, test.template)
			}
			var sb strings.Builder
			enc := e.EncoderTo(&sb)
			var execErr error
			for _, row := range test.rows {
				if execErr = enc.Encode(row); execErr != nil {
					break
				}
			}
			if (execErr != nil) != test.execErr {
				t.Fatalf("Encode: %v; want error: %t", execErr, test.execErr)
			}
			if test.errText != "" && execErr.Error() != test.errText {
				t.Errorf(
					"Encode: %q; want %q",
					execErr.Error(), test.errText,
				)
			}
			if err := enc.Flush(); err != nil {
				t.Fatalf("Flush: %v", err)
			}
			if act := sb.String(); act != test.exp {
				t.Errorf(
					"output: %q; want %q",
					act, test.exp,
				)
			}
		})
	}
}
