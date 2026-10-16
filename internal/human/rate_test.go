package human

import (
	"testing"
	"time"
)

func TestParseRate(t *testing.T) {
	for _, test := range []struct {
		name string
		s    string
		exp  time.Duration
		err  bool
	}{
		{
			name: "per second",
			s:    "250/s",
			exp:  4 * time.Millisecond,
		},
		{
			name: "one per second",
			s:    "1/s",
			exp:  time.Second,
		},
		{
			name: "per minute",
			s:    "60/m",
			exp:  time.Second,
		},
		{
			name: "per hour",
			s:    "2/h",
			exp:  30 * time.Minute,
		},
		{
			name: "per millisecond",
			s:    "4/ms",
			exp:  250 * time.Microsecond,
		},
		{
			name: "per several seconds",
			s:    "1000/4s",
			exp:  4 * time.Millisecond,
		},
		{
			name: "per compound period",
			s:    "3/1m30s",
			exp:  30 * time.Second,
		},
		{
			name: "one per interval",
			s:    "1/7s",
			exp:  7 * time.Second,
		},
		{
			name: "inexact",
			s:    "3/s",
			err:  true,
		},
		{
			name: "zero count",
			s:    "0/s",
			err:  true,
		},
		{
			name: "negative count",
			s:    "-1/s",
			err:  true,
		},
		{
			name: "zero period",
			s:    "1/0s",
			err:  true,
		},
		{
			name: "no period",
			s:    "250",
			err:  true,
		},
		{
			name: "unknown period",
			s:    "250/d",
			err:  true,
		},
		{
			name: "empty",
			s:    "",
			err:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			act, err := ParseRate(test.s)
			if test.err && err == nil {
				t.Errorf("want error; got nothing")
			}
			if !test.err && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if exp := test.exp; act != exp {
				t.Errorf("interval: %v; want %v", act, exp)
			}
		})
	}
}

func TestFormatRate(t *testing.T) {
	for _, test := range []struct {
		name string
		d    time.Duration
		exp  string
	}{
		{
			name: "per second",
			d:    4 * time.Millisecond,
			exp:  "250/s",
		},
		{
			name: "one per second",
			d:    time.Second,
			exp:  "1/s",
		},
		{
			name: "sub millisecond",
			d:    250 * time.Microsecond,
			exp:  "4000/s",
		},
		{
			name: "per minute",
			d:    time.Minute,
			exp:  "1/m",
		},
		{
			name: "per hour",
			d:    30 * time.Minute,
			exp:  "2/h",
		},
		{
			name: "divides nothing",
			d:    7 * time.Second,
			exp:  "1/7s",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			act := FormatRate(test.d)
			if exp := test.exp; act != exp {
				t.Errorf("format: %q; want %q", act, exp)
			}
			back, err := ParseRate(act)
			if err != nil {
				t.Errorf("parse back: %v", err)
			}
			if exp := test.d; back != exp {
				t.Errorf("round trip: %v; want %v", back, exp)
			}
		})
	}
}
