package main

import (
	"io"
	"os"

	"golang.org/x/term"
)

// Output is coloured from the terminal's own palette, because only the
// user's theme knows the background: a fixed xterm-256 value that reads
// well on black is faint on white.
const (
	// keyStyle is the SGR sequence for a text key and a json key: bright
	// blue.
	keyStyle = "\x1b[94m"
	// pathStyle is keyStyle, faint, for the path of a nested key ("stats."
	// of "stats.visible"), so the leaves stand out and the groups read as
	// groups.
	pathStyle = "\x1b[2;94m"
	// stringStyle is the SGR sequence for a json string value: yellow.
	stringStyle = "\x1b[33m"
)

// isTerminal reports whether w is a terminal.
func isTerminal(w io.Writer) bool {
	f, ok := w.(interface {
		Fd() uintptr
	})
	return ok && term.IsTerminal(int(f.Fd()))
}

// colorEnabled reports whether the environment allows colour: no NO_COLOR,
// and a TERM that is not dumb.
func colorEnabled() bool {
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	return os.Getenv("TERM") != "dumb"
}
