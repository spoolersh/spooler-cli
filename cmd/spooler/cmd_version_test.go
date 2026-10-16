package main

import (
	"testing"

	"github.com/spoolersh/spooler-go"
)

func TestVersion(t *testing.T) {
	// A test binary has no module version, so the CLI reports devel.
	exp := result{
		stdout: record(t, versionOutput{
			CLI: "devel",
			SDK: spooler.Version,
		}),
	}
	expect(t, run(t, &fakeClient{}, "", "version"), exp)
	// The flag is the version command, whatever follows it.
	expect(t, run(t, &fakeClient{}, "", "--version", "stats"), exp)
}
