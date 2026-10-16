package json

import (
	"strings"
	"testing"
	"time"
)

type testRow struct {
	Name      string    `json:"name"`
	Stats     testStats `json:"stats"`
	Tags      []string  `json:"tags"`
	Empty     struct{}  `json:"empty"`
	None      []int     `json:"none"`
	Ratio     float64   `json:"ratio"`
	Active    bool      `json:"active"`
	Note      *string   `json:"note"`
	CreatedAt time.Time `json:"createdAt,omitzero"`
}

type testStats struct {
	Visible int64   `json:"visible"`
	Points  []int64 `json:"points"`
}

var rows = []testRow{
	{
		Name:      "orders \"quoted\"",
		Stats:     testStats{Visible: 12, Points: []int64{1, 2}},
		Tags:      []string{"a", "b"},
		None:      []int{},
		Ratio:     0.5,
		Active:    true,
		CreatedAt: time.Date(2026, 10, 15, 9, 30, 0, 0, time.UTC),
	},
	{
		Name: "refunds",
	},
}

const (
	keyStyle    = "\x1b[38;5;75m"
	stringStyle = "\x1b[32m"
)

func TestEncoder(t *testing.T) {
	for _, test := range []struct {
		name   string
		indent string
		exp    string
	}{
		{
			name: "one line per document",
			exp: `{"name":"orders \"quoted\"","stats":{"visible":12,"points":[1,2]},"tags":["a","b"],"empty":{},"none":[],"ratio":0.5,"active":true,"note":null,"createdAt":"2026-10-15T09:30:00Z"}` + "\n" +
				`{"name":"refunds","stats":{"visible":0,"points":null},"tags":null,"empty":{},"none":null,"ratio":0,"active":false,"note":null}` + "\n",
		},
		{
			name:   "indented",
			indent: "  ",
			exp: strings.Join([]string{
				"{",
				`  "name": "orders \"quoted\"",`,
				`  "stats": {`,
				`    "visible": 12,`,
				`    "points": [`,
				`      1,`,
				`      2`,
				`    ]`,
				`  },`,
				`  "tags": [`,
				`    "a",`,
				`    "b"`,
				`  ],`,
				`  "empty": {},`,
				`  "none": [],`,
				`  "ratio": 0.5,`,
				`  "active": true,`,
				`  "note": null,`,
				`  "createdAt": "2026-10-15T09:30:00Z"`,
				"}",
				"{",
				`  "name": "refunds",`,
				`  "stats": {`,
				`    "visible": 0,`,
				`    "points": null`,
				`  },`,
				`  "tags": null,`,
				`  "empty": {},`,
				`  "none": null,`,
				`  "ratio": 0,`,
				`  "active": false,`,
				`  "note": null`,
				"}",
				"",
			}, "\n"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			act := encode(t, Encoding[testRow]{
				Indent: test.indent,
			})
			if act != test.exp {
				t.Errorf(
					"json:\n%s\nwant:\n%s",
					act, test.exp,
				)
			}
		})
	}
}

// TestStyledEncoderLayout pins that styles add escape sequences and nothing
// else: stripped of them, the output is byte for byte what encoding/json
// writes with the same indent.
func TestStyledEncoderLayout(t *testing.T) {
	for _, indent := range []string{"", "  "} {
		t.Run("indent "+quote(indent), func(t *testing.T) {
			plain := encode(t, Encoding[testRow]{
				Indent: indent,
			})
			styled := encode(t, Encoding[testRow]{
				Indent:      indent,
				KeyStyle:    keyStyle,
				StringStyle: stringStyle,
			})
			if act := Strip(styled); act != plain {
				t.Errorf(
					"stripped:\n%s\nwant:\n%s",
					act, plain,
				)
			}
		})
	}
}

func TestStyledEncoderStyles(t *testing.T) {
	for _, test := range []struct {
		name        string
		keyStyle    string
		stringStyle string
		expKey      string
		expString   string
	}{
		{
			name:      "keys only",
			keyStyle:  keyStyle,
			expKey:    keyStyle + `"name"` + reset + `:`,
			expString: `:"refunds"`,
		},
		{
			name:        "keys and strings",
			keyStyle:    keyStyle,
			stringStyle: stringStyle,
			expKey:      keyStyle + `"name"` + reset + `:`,
			expString:   `:` + stringStyle + `"refunds"` + reset,
		},
		{
			name:        "strings only",
			stringStyle: stringStyle,
			expKey:      `"name":`,
			expString:   `:` + stringStyle + `"refunds"` + reset,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := Encoding[testRow]{
				KeyStyle:    test.keyStyle,
				StringStyle: test.stringStyle,
			}
			var sb strings.Builder
			enc := e.EncoderTo(&sb)
			if err := enc.Encode(rows[1]); err != nil {
				t.Fatalf("Encode: %v", err)
			}
			act := sb.String()
			if !strings.Contains(act, test.expKey) {
				t.Errorf(
					"json %q lacks key %q",
					act, test.expKey,
				)
			}
			if !strings.Contains(act, test.expString) {
				t.Errorf(
					"json %q lacks string %q",
					act, test.expString,
				)
			}
		})
	}
}

func TestStrip(t *testing.T) {
	for _, test := range []struct {
		name string
		s    string
		exp  string
	}{
		{
			name: "empty",
			s:    "",
			exp:  "",
		},
		{
			name: "plain",
			s:    "plain",
			exp:  "plain",
		},
		{
			name: "styled key",
			s:    keyStyle + `"a"` + reset + ": 1",
			exp:  `"a": 1`,
		},
		{
			name: "faint path",
			s:    "\x1b[2;38;5;75mstats.\x1b[0m",
			exp:  "stats.",
		},
		{
			name: "unterminated sequence is dropped",
			s:    "cut \x1b[38;5",
			exp:  "cut ",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if act := Strip(test.s); act != test.exp {
				t.Errorf("Strip(%q): %q; want %q", test.s, act, test.exp)
			}
		})
	}
}

// encode renders rows through e and returns the output.
func encode(t *testing.T, e Encoding[testRow]) string {
	t.Helper()
	var sb strings.Builder
	enc := e.EncoderTo(&sb)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}
	if err := enc.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	return sb.String()
}

// quote wraps an indent in quotes for a test name.
func quote(s string) string {
	return `"` + s + `"`
}
