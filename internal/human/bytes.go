// Package human converts values to and from the forms people write.
package human

import (
	"fmt"
	"strconv"
	"strings"
)

// Byte units are the IEC binary prefixes only: no KB/MB, no ambiguity.
const (
	B   uint64 = 1
	KiB        = B << 10
	MiB        = KiB << 10
	GiB        = MiB << 10
	TiB        = GiB << 10
	PiB        = TiB << 10
)

var byteUnits = []struct {
	name string
	size uint64
}{
	{"PiB", PiB},
	{"TiB", TiB},
	{"GiB", GiB},
	{"MiB", MiB},
	{"KiB", KiB},
	{"B", B},
}

// maxFraction bounds the digits after the point in either direction.
const maxFraction = 3

// ParseBytes reads a byte count written as a plain integer or a number with
// a binary unit (64KiB, 1.5GiB, 512 MiB). A fraction must come out as
// whole bytes.
func ParseBytes(s string) (uint64, error) {
	i := strings.IndexFunc(s, func(r rune) bool {
		return r != '.' && (r < '0' || r > '9')
	})
	num, unit := s, ""
	if i >= 0 {
		num, unit = s[:i], strings.TrimSpace(s[i:])
	}
	size := B
	if unit != "" {
		var ok bool
		if size, ok = unitSize(unit); !ok {
			return 0, fmt.Errorf(
				"bytes %q: unknown unit %q",
				s, unit,
			)
		}
	}
	whole, frac, dot := strings.Cut(num, ".")
	if whole == "" || (dot && frac == "") || len(frac) > maxFraction {
		return 0, fmt.Errorf(
			"bytes %q: malformed number",
			s,
		)
	}
	w, err := strconv.ParseUint(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf(
			"bytes %q: %w",
			s, err,
		)
	}
	n, ok := mul(w, size)
	if !ok {
		return 0, fmt.Errorf(
			"bytes %q: too big",
			s,
		)
	}
	if frac == "" {
		return n, nil
	}
	f, err := strconv.ParseUint(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf(
			"bytes %q: %w",
			s, err,
		)
	}
	scale := pow10(len(frac))
	if (f*size)%scale != 0 {
		return 0, fmt.Errorf(
			"bytes %q: not a whole number of bytes",
			s,
		)
	}
	n, ok = add(n, f*size/scale)
	if !ok {
		return 0, fmt.Errorf(
			"bytes %q: too big",
			s,
		)
	}
	return n, nil
}

// FormatBytes writes n exactly, in the largest binary unit that needs at
// most three fraction digits: 65536 is 64KiB, 1610612736 is 1.5GiB. What
// no unit expresses that way is written in bytes. ParseBytes reads the
// result back to n.
func FormatBytes(n uint64) string {
	for _, u := range byteUnits {
		if u.size == B || n < u.size {
			continue
		}
		whole, rem := n/u.size, n%u.size
		if rem == 0 {
			return fmt.Sprintf(
				"%d%s",
				whole, u.name,
			)
		}
		// rem/size as a decimal fraction, if it terminates within
		// maxFraction digits.
		scale := pow10(maxFraction)
		if (rem*scale)%u.size != 0 {
			continue
		}
		frac := strconv.FormatUint(rem*scale/u.size, 10)
		frac = strings.Repeat("0", maxFraction-len(frac)) + frac
		frac = strings.TrimRight(frac, "0")
		return fmt.Sprintf(
			"%d.%s%s",
			whole, frac, u.name,
		)
	}
	return fmt.Sprintf(
		"%dB",
		n,
	)
}

func unitSize(name string) (uint64, bool) {
	for _, u := range byteUnits {
		if u.name == name {
			return u.size, true
		}
	}
	return 0, false
}

func pow10(n int) uint64 {
	p := uint64(1)
	for range n {
		p *= 10
	}
	return p
}

func mul(a, b uint64) (uint64, bool) {
	if a != 0 && b > ^uint64(0)/a {
		return 0, false
	}
	return a * b, true
}

func add(a, b uint64) (uint64, bool) {
	if b > ^uint64(0)-a {
		return 0, false
	}
	return a + b, true
}
