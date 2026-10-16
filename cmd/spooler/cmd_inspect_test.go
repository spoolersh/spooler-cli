package main

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spoolersh/spooler-go"
)

// inspected is what the fake inspect returns: m1, failed after three
// retries, with a five-byte payload, under retention.
func inspected() spooler.InspectResult {
	return spooler.InspectResult{
		ID:         "m1",
		State:      spooler.MessageStateFailed,
		Retries:    3,
		CreatedAt:  time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC),
		ExpiresAt:  time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
		DataStream: io.NopCloser(strings.NewReader("hello")),
		DataSize:   5,
	}
}

// inspectedOutput is inspected() as the CLI prints it.
func inspectedOutput() inspectResultOutput {
	return inspectResultOutput{
		ID:        "m1",
		State:     "failed",
		Retries:   3,
		CreatedAt: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
		DataSize:  5,
	}
}

func TestInspect(t *testing.T) {
	inspectRecord := record(t, inspectedOutput())
	for _, test := range []struct {
		name      string
		args      []string
		expHandle string
		exp       result
	}{
		{
			name:      "payload alone",
			args:      []string{"inspect", "-s", "main", "--handle", "H1"},
			expHandle: "H1",
			exp: result{
				stdout: "hello",
			},
		},
		{
			name:      "include meta",
			args:      []string{"inspect", "-s", "main", "--handle", "H1", "-i"},
			expHandle: "H1",
			exp: result{
				stdout: inspectRecord + "\nhello",
			},
		},
		{
			name: "output without include meta",
			args: []string{"inspect", "-s", "main", "--handle", "H1", "-o", "text"},
			exp: result{
				stderr: "--output takes effect only with --include-meta\n",
				code:   2,
			},
		},
		{
			name: "human with json",
			args: []string{"inspect", "-s", "main", "--handle", "H1", "-i", "--human"},
			exp: result{
				stderr: "--human takes effect only with `-o table` or `-o text`\n",
				code:   2,
			},
		},
		{
			name: "compact without include meta",
			args: []string{"inspect", "-s", "main", "--handle", "H1", "--compact"},
			exp: result{
				stderr: "--compact takes effect only with --include-meta\n",
				code:   2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var handle string
			fake := &fakeClient{
				DoInspect: func(_ context.Context, r spooler.InspectRequest) (spooler.InspectResult, error) {
					handle = string(r.Handle)
					if r.Spool != "main" {
						t.Errorf("spool: %q; want %q", r.Spool, "main")
					}
					return inspected(), nil
				},
			}
			act := run(t, fake, "", test.args...)
			expect(t, act, test.exp)
			if handle != test.expHandle {
				t.Errorf("handle: %q; want %q", handle, test.expHandle)
			}
		})
	}
}

func TestInspectFiles(t *testing.T) {
	dir := t.TempDir()
	metaFile := filepath.Join(dir, "meta.json")
	dataFile := filepath.Join(dir, "payload.bin")
	fake := &fakeClient{
		DoInspect: func(context.Context, spooler.InspectRequest) (spooler.InspectResult, error) {
			return inspected(), nil
		},
	}
	act := run(t, fake, "", "inspect", "-s", "main", "--handle", "H1", "--meta-file", metaFile, "-D", dataFile)
	expect(t, act, result{})
	meta, mode := readFile(t, metaFile)
	exp := frames(t, inspectedOutput())
	if meta != exp {
		t.Errorf("meta file:\n%s\nwant:\n%s", meta, exp)
	}
	if mode != 0644 && mode != 0666 {
		t.Errorf("meta file mode: %o; want 0666 less the umask", mode)
	}
	data, _ := readFile(t, dataFile)
	if data != "hello" {
		t.Errorf("data file: %q; want %q", data, "hello")
	}
}
