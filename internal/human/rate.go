package human

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ratePeriods are the periods FormatRate tries, in order of preference.
var ratePeriods = []struct {
	name string
	d    time.Duration
}{
	{"s", time.Second},
	{"m", time.Minute},
	{"h", time.Hour},
}

// ParseRate reads a rate written as a count per period, 250/s, 60/m or
// 1000/4s, and returns the interval between two events. The period is a
// duration; a bare unit means one of it. The interval must come out as
// whole nanoseconds.
func ParseRate(s string) (time.Duration, error) {
	count, period, ok := strings.Cut(s, "/")
	if !ok {
		return 0, fmt.Errorf(
			"rate %q: want <count>/<period>",
			s,
		)
	}
	n, err := strconv.ParseInt(count, 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf(
			"rate %q: count must be a positive integer",
			s,
		)
	}
	if period != "" && (period[0] < '0' || period[0] > '9') {
		period = "1" + period
	}
	d, err := time.ParseDuration(period)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf(
			"rate %q: period must be a positive duration",
			s,
		)
	}
	if int64(d)%n != 0 {
		return 0, fmt.Errorf(
			"rate %q: not a whole number of nanoseconds",
			s,
		)
	}
	return d / time.Duration(n), nil
}

// FormatRate writes an interval as a count per period: per second when
// the interval divides a second exactly (4ms is 250/s), else per minute
// or hour, else as one per the interval itself (1/7s). ParseRate reads
// every form back.
func FormatRate(interval time.Duration) string {
	if interval > 0 {
		for _, p := range ratePeriods {
			if p.d >= interval && p.d%interval == 0 {
				return fmt.Sprintf(
					"%d/%s",
					p.d/interval, p.name,
				)
			}
		}
	}
	return fmt.Sprintf(
		"1/%s",
		interval,
	)
}
