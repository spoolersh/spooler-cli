package main

import "cmp"

// flagRank sorts a command's flags in help by meaning rather than alphabet.
var flagRank = map[string]int{
	// Root.
	"config":    1,
	"profile":   2,
	"host":      3,
	"api-key":   4,
	"no-tls":    5,
	"log-level": 6,
	"debug":     7,
	"version":   8,

	// What the command works on.
	"spool": 10,
	"queue": 11,
	"file":  12,
	"uid":   13,

	// What to act on.
	"handle":     20,
	"lease":      21,
	"lease-file": 22,

	// How long to wait, or to delay.
	"wait":  30,
	"delay": 31,

	// What to do with the result, and the consent it needs.
	"ack":  40,
	"mode": 41,
	"yes":  42,

	// The record and the payload.
	"include-meta": 50,
	"meta-file":    51,
	"data":         52,
	"data-file":    53,
	"overwrite":    54,

	// The send's dedup key and the handle it may return.
	"dedup-string":  55,
	"dedup-hash":    56,
	"return-handle": 57,

	// A queue's settings, in the docs' order.
	"lease-timeout":   60,
	"max-retries":     61,
	"dedup-window":    62,
	"retention":       63,
	"recv-rate-limit": 64,
	"recv-burst":      65,

	// What a listing keeps.
	"state": 70,
	"match": 71,
	"limit": 72,

	// Output, last.
	"output":   90,
	"columns":  91,
	"template": 92,
	"compact":  93,
	"human":    94,
	"no-color": 95,
}

// compareFlags orders two flag names: both ranked, by rank; one ranked, that
// one first; neither, alphabetically.
func compareFlags(flag0, flag1 string) int {
	r0, ok0 := flagRank[flag0]
	r1, ok1 := flagRank[flag1]
	switch {
	case ok0 && ok1:
		return cmp.Compare(r0, r1)
	case ok0:
		return -1
	case ok1:
		return 1
	}
	return 0
}
