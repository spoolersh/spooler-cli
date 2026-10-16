package human

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	for _, test := range []struct {
		name string
		s    string
		exp  time.Duration
		err  bool
	}{
		{
			name: "seconds",
			s:    "90s",
			exp:  90 * time.Second,
		},
		{
			name: "two units",
			s:    "1h30m",
			exp:  90 * time.Minute,
		},
		{
			name: "days",
			s:    "1d12h",
			exp:  36 * time.Hour,
		},
		{
			name: "three units",
			s:    "1h1m1s",
			exp:  time.Hour + time.Minute + time.Second,
		},
		{
			name: "millis",
			s:    "250ms",
			exp:  250 * time.Millisecond,
		},
		{
			name: "micros",
			s:    "1µs",
			exp:  time.Microsecond,
		},
		{
			name: "micros ascii",
			s:    "1us",
			exp:  time.Microsecond,
		},
		{
			name: "micros greek mu",
			s:    "1μs",
			exp:  time.Microsecond,
		},
		{
			name: "nanos",
			s:    "7ns",
			exp:  7 * time.Nanosecond,
		},
		{
			name: "fraction",
			s:    "1.5h",
			exp:  90 * time.Minute,
		},
		{
			name: "fraction of seconds",
			s:    "1.5s",
			exp:  1500 * time.Millisecond,
		},
		{
			name: "spaced",
			s:    "1 h",
			exp:  time.Hour,
		},
		{
			name: "zero",
			s:    "0",
			exp:  0,
		},
		{
			name: "negative",
			s:    "-1m30s",
			exp:  -90 * time.Second,
		},
		{
			name: "fraction of nanoseconds",
			s:    "1.5ns",
			err:  true,
		},
		{
			name: "too many digits",
			s:    "1.0001s",
			err:  true,
		},
		{
			name: "missing unit",
			s:    "90",
			err:  true,
		},
		{
			name: "missing number",
			s:    "h",
			err:  true,
		},
		{
			name: "trailing number",
			s:    "1h30",
			err:  true,
		},
		{
			name: "week",
			s:    "1w",
			err:  true,
		},
		{
			name: "trailing dot",
			s:    "1.h",
			err:  true,
		},
		{
			name: "too big",
			s:    "200000000000d",
			err:  true,
		},
		{
			name: "empty",
			s:    "",
			err:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			act, err := ParseDuration(test.s)
			if test.err && err == nil {
				t.Errorf("want error; got nothing")
			}
			if !test.err && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if exp := test.exp; act != exp {
				t.Errorf("duration: %v; want %v", act, exp)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	for _, test := range []struct {
		name string
		d    time.Duration
		exp  string
		// lossy marks a duration that needs more than two units, so it
		// does not read back to itself.
		lossy bool
	}{
		{
			name: "zero",
			d:    0,
			exp:  "0s",
		},
		{
			name: "seconds",
			d:    45 * time.Second,
			exp:  "45s",
		},
		{
			name: "minutes and seconds",
			d:    90 * time.Second,
			exp:  "1m30s",
		},
		{
			name: "hours and minutes",
			d:    90 * time.Minute,
			exp:  "1h30m",
		},
		{
			name: "days and hours",
			d:    36 * time.Hour,
			exp:  "1d12h",
		},
		{
			name: "whole day",
			d:    24 * time.Hour,
			exp:  "1d",
		},
		{
			name: "many days",
			d:    400 * 24 * time.Hour,
			exp:  "400d",
		},
		{
			name: "millis",
			d:    250 * time.Millisecond,
			exp:  "250ms",
		},
		{
			name:  "millis dropped past a second",
			d:     1500 * time.Millisecond,
			exp:   "1s",
			lossy: true,
		},
		{
			name:  "millis dropped past a minute",
			d:     time.Minute + 10*time.Second + 500*time.Millisecond,
			exp:   "1m10s",
			lossy: true,
		},
		{
			name:  "one unit below a second",
			d:     1500 * time.Nanosecond,
			exp:   "1µs",
			lossy: true,
		},
		{
			name:  "one unit below a second, millis",
			d:     10500 * time.Microsecond,
			exp:   "10ms",
			lossy: true,
		},
		{
			name: "negative",
			d:    -90 * time.Second,
			exp:  "-1m30s",
		},
		{
			name:  "third unit dropped",
			d:     time.Hour + time.Minute + time.Second,
			exp:   "1h1m",
			lossy: true,
		},
		{
			name:  "gap unit dropped",
			d:     time.Hour + time.Second,
			exp:   "1h",
			lossy: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			act := FormatDuration(test.d)
			if exp := test.exp; act != exp {
				t.Errorf("format: %q; want %q", act, exp)
			}
			back, err := ParseDuration(act)
			if err != nil {
				t.Errorf("parse back: %v", err)
			}
			if exp := test.d; !test.lossy && back != exp {
				t.Errorf("round trip: %v; want %v", back, exp)
			}
		})
	}
}
