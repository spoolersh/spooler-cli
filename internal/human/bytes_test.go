package human

import (
	"testing"
)

func TestParseBytes(t *testing.T) {
	for _, test := range []struct {
		name string
		s    string
		exp  uint64
		err  bool
	}{
		{
			name: "plain",
			s:    "65536",
			exp:  65536,
		},
		{
			name: "bytes unit",
			s:    "512B",
			exp:  512,
		},
		{
			name: "kib",
			s:    "64KiB",
			exp:  65536,
		},
		{
			name: "spaced",
			s:    "64 KiB",
			exp:  65536,
		},
		{
			name: "fraction",
			s:    "1.5GiB",
			exp:  1610612736,
		},
		{
			name: "three digits",
			s:    "1.125MiB",
			exp:  1179648,
		},
		{
			name: "pib",
			s:    "1PiB",
			exp:  1 << 50,
		},
		{
			name: "fraction of bytes",
			s:    "1.5B",
			err:  true,
		},
		{
			name: "inexact fraction",
			s:    "1.3KiB",
			err:  true,
		},
		{
			name: "too many digits",
			s:    "1.0001GiB",
			err:  true,
		},
		{
			name: "decimal unit",
			s:    "64KB",
			err:  true,
		},
		{
			name: "lowercase unit",
			s:    "64kib",
			err:  true,
		},
		{
			name: "trailing dot",
			s:    "64.KiB",
			err:  true,
		},
		{
			name: "negative",
			s:    "-1KiB",
			err:  true,
		},
		{
			name: "too big",
			s:    "20000000PiB",
			err:  true,
		},
		{
			name: "empty",
			s:    "",
			err:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			act, err := ParseBytes(test.s)
			if test.err && err == nil {
				t.Errorf("want error; got nothing")
			}
			if !test.err && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if exp := test.exp; act != exp {
				t.Errorf("bytes: %d; want %d", act, exp)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	for _, test := range []struct {
		name string
		n    uint64
		exp  string
	}{
		{
			name: "zero",
			n:    0,
			exp:  "0B",
		},
		{
			name: "bytes",
			n:    512,
			exp:  "512B",
		},
		{
			name: "kib",
			n:    65536,
			exp:  "64KiB",
		},
		{
			name: "gib fraction",
			n:    1610612736,
			exp:  "1.5GiB",
		},
		{
			name: "mib fraction",
			n:    1638400000,
			exp:  "1562.5MiB",
		},
		{
			name: "three digits",
			n:    1179648,
			exp:  "1.125MiB",
		},
		{
			name: "odd bytes",
			n:    1025,
			exp:  "1025B",
		},
		{
			name: "pib",
			n:    1 << 50,
			exp:  "1PiB",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			act := FormatBytes(test.n)
			if exp := test.exp; act != exp {
				t.Errorf("format: %q; want %q", act, exp)
			}
			back, err := ParseBytes(act)
			if err != nil {
				t.Errorf("parse back: %v", err)
			}
			if exp := test.n; back != exp {
				t.Errorf("round trip: %d; want %d", back, exp)
			}
		})
	}
}
