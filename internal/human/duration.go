package human

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// durationUnits are the units FormatDuration writes and ParseDuration reads,
// largest first. A day is 24 hours; there is no week or year, whose lengths
// vary.
var durationUnits = []struct {
	name string
	d    time.Duration
}{
	{"d", 24 * time.Hour},
	{"h", time.Hour},
	{"m", time.Minute},
	{"s", time.Second},
	{"ms", time.Millisecond},
	{"µs", time.Microsecond},
	{"ns", time.Nanosecond},
}

// ParseDuration reads a duration written as numbers with units (90s, 1h30m,
// 1d12h, 250ms, 1.5h), a bare 0, or any of those with a leading minus. The
// units are d, h, m, s, ms, µs (spelt as time.ParseDuration takes it: us,
// µs or μs) and ns; a fraction must come out as whole nanoseconds. A bare
// number other than 0 has no unit and is an error.
func ParseDuration(s string) (time.Duration, error) {
	rest, neg := strings.CutPrefix(s, "-")
	if rest == "0" {
		return 0, nil
	}
	var total uint64
	for n := 0; rest != "" || n == 0; n++ {
		i := strings.IndexFunc(rest, func(r rune) bool {
			return r != '.' && (r < '0' || r > '9')
		})
		if i == 0 || rest == "" {
			return 0, fmt.Errorf(
				"duration %q: want a number followed by a unit, as 1h30m",
				s,
			)
		}
		if i < 0 {
			return 0, fmt.Errorf(
				"duration %q: missing unit, as in %ss",
				s, rest,
			)
		}
		num, unit := rest[:i], rest[i:]
		if j := strings.IndexFunc(unit, func(r rune) bool {
			return r == '.' || ('0' <= r && r <= '9')
		}); j >= 0 {
			unit, rest = unit[:j], unit[j:]
		} else {
			rest = ""
		}
		size, ok := durationUnit(strings.TrimSpace(unit))
		if !ok {
			return 0, fmt.Errorf(
				"duration %q: unknown unit %q",
				s, strings.TrimSpace(unit),
			)
		}
		v, err := parseScaled(num, uint64(size))
		if err != nil {
			return 0, fmt.Errorf(
				"duration %q: %w",
				s, err,
			)
		}
		if total, ok = add(total, v); !ok || total > math.MaxInt64 {
			return 0, fmt.Errorf(
				"duration %q: too big",
				s,
			)
		}
	}
	d := time.Duration(total)
	if neg {
		d = -d
	}
	return d, nil
}

// FormatDuration writes d in at most two adjacent units, the largest that
// fits and the one below it, down to the second and never past it: 90s is
// 1m30s, 36h is 1d12h, 1h1m1s is 1h1m, 1h1s is 1h and 19.017s is 19s. Below
// a second it is one unit, the largest that fits: 10.5ms is 10ms. Zero is
// 0s and a negative duration has a leading minus. ParseDuration reads the
// result back, to d when nothing was dropped.
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	n, sign := uint64(d), ""
	if d < 0 {
		n, sign = -n, "-"
	}
	for i, u := range durationUnits {
		size := uint64(u.d)
		if n < size {
			continue
		}
		whole, rem := n/size, n%size
		// The unit below joins only when it is a second or more.
		if u.d > time.Second {
			next := durationUnits[i+1]
			if rem >= uint64(next.d) {
				return fmt.Sprintf(
					"%s%d%s%d%s",
					sign, whole, u.name, rem/uint64(next.d), next.name,
				)
			}
		}
		return fmt.Sprintf(
			"%s%d%s",
			sign, whole, u.name,
		)
	}
	return sign + "0s" // Unreachable: the last unit is a nanosecond.
}

func durationUnit(name string) (time.Duration, bool) {
	// The spellings time.ParseDuration takes: ASCII, and the Greek mu
	// (U+03BC) for the micro sign (U+00B5).
	if name == "us" || name == "μs" {
		name = "µs"
	}
	for _, u := range durationUnits {
		if u.name == name {
			return u.d, true
		}
	}
	return 0, false
}

// parseScaled reads a number with at most maxFraction fraction digits and
// returns it times size, which must come out whole.
func parseScaled(num string, size uint64) (uint64, error) {
	whole, frac, dot := strings.Cut(num, ".")
	if whole == "" || (dot && frac == "") || len(frac) > maxFraction {
		return 0, fmt.Errorf(
			"malformed number %q",
			num,
		)
	}
	w, err := strconv.ParseUint(whole, 10, 64)
	if err != nil {
		return 0, err
	}
	n, ok := mul(w, size)
	if !ok {
		return 0, fmt.Errorf("too big")
	}
	if frac == "" {
		return n, nil
	}
	f, err := strconv.ParseUint(frac, 10, 64)
	if err != nil {
		return 0, err
	}
	scale := pow10(len(frac))
	if (f*size)%scale != 0 {
		return 0, fmt.Errorf(
			"%q is not a whole number of the base unit",
			num,
		)
	}
	n, ok = add(n, f*size/scale)
	if !ok {
		return 0, fmt.Errorf("too big")
	}
	return n, nil
}
