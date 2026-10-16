package fields

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type testRow struct {
	Name      string `json:"name"`
	Skipped   string `json:"-"`
	Untagged  string
	CreatedAt time.Time `json:"createdAt"`
	Tags      []string  `json:"tags"`
	Stats     testStats `json:"stats"`
}

type testStats struct {
	Visible   int64     `json:"visible"`
	SampledAt time.Time `json:"sampledAt,omitzero"`
	Pending   int64     `json:"pending,omitempty"`
}

func TestOf(t *testing.T) {
	exp := []string{
		"name",
		"createdAt",
		"stats.visible",
		"stats.sampledAt",
		"stats.pending",
	}
	act := Paths(Of(testRow{}))
	if diff := cmp.Diff(exp, act); diff != "" {
		t.Errorf("paths:\n%s", diff)
	}
}

type testHuman struct {
	CreatedAt time.Time `json:"createdAt" human:"age"`
	ExpiresAt time.Time `json:"expiresAt" human:"age"`
	SampledAt time.Time `json:"sampledAt" human:"age"`
	Timeout   int       `json:"timeout" human:"seconds"`
	Interval  int64     `json:"interval" human:"rate"`
	NoLimit   int64     `json:"noLimit" human:"rate"`
	Size      uint64    `json:"size" human:"bytes"`
	Plain     int       `json:"plain"`
	Wrong     string    `json:"wrong" human:"bytes"`
}

func TestHuman(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	fs := Of(testHuman{
		CreatedAt: now.Add(-90 * time.Minute),
		ExpiresAt: now.Add(45 * time.Second),
		SampledAt: now.Add(-19*time.Second - 17*time.Millisecond),
		Timeout:   36 * 3600,
		Interval:  4000,
		Size:      1536 << 10,
		Plain:     7,
		Wrong:     "x",
	})
	for _, test := range []struct {
		name string
		path string
		exp  string
		ok   bool
		err  bool
	}{
		{
			name: "past time as age",
			path: "createdAt",
			exp:  "1h30m",
			ok:   true,
		},
		{
			name: "future time as age",
			path: "expiresAt",
			exp:  "45s",
			ok:   true,
		},
		{
			name: "age in whole seconds",
			path: "sampledAt",
			exp:  "19s",
			ok:   true,
		},
		{
			name: "seconds as duration",
			path: "timeout",
			exp:  "1d12h",
			ok:   true,
		},
		{
			name: "interval as rate",
			path: "interval",
			exp:  "250/s",
			ok:   true,
		},
		{
			name: "zero interval is unlimited",
			path: "noLimit",
			exp:  "unlimited",
			ok:   true,
		},
		{
			name: "bytes in binary units",
			path: "size",
			exp:  "1.5MiB",
			ok:   true,
		},
		{
			name: "untagged field is left alone",
			path: "plain",
		},
		{
			name: "tag on a string is an error",
			path: "wrong",
			err:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var f Field
			for _, g := range fs {
				if g.Path == test.path {
					f = g
				}
			}
			act, ok, err := Human(f, now)
			if test.err && err == nil {
				t.Errorf("want error; got nothing")
			}
			if !test.err && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if ok != test.ok {
				t.Errorf("ok: %t; want %t", ok, test.ok)
			}
			if act != test.exp {
				t.Errorf("human: %q; want %q", act, test.exp)
			}
		})
	}
}

func TestOmitted(t *testing.T) {
	for _, test := range []struct {
		name string
		row  testRow
		exp  []bool
	}{
		{
			name: "zero",
			row:  testRow{},
			exp:  []bool{false, false, false, true, true},
		},
		{
			name: "set",
			row: testRow{
				Stats: testStats{
					SampledAt: time.Date(2026, 10, 12, 9, 31, 0, 0, time.UTC),
					Pending:   1,
				},
			},
			exp: []bool{false, false, false, false, false},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fs := Of(test.row)
			act := make([]bool, len(fs))
			for i, f := range fs {
				act[i] = Omitted(f)
			}
			if diff := cmp.Diff(test.exp, act); diff != "" {
				t.Errorf("omitted:\n%s", diff)
			}
		})
	}
}
