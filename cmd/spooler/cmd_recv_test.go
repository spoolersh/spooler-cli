package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"testing/iotest"
	"time"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-go"
)

// TestRecvInterrupted covers a signal landing mid-payload: the lease is
// released, and the runner exits with the signal's code in silence unless
// the release failed or ended unknown, when the message names the lease.
func TestRecvInterrupted(t *testing.T) {
	for _, test := range []struct {
		name    string
		release error
		exp     result
	}{
		{
			name: "released",
			exp: result{
				code: 130,
			},
		},
		{
			name:    "release failed",
			release: apiError(spooler.ErrorKindRateLimited, "slow down"),
			exp: result{
				stderr: "write payload: context canceled; release: slow down" +
					" (lease \"L1\")\n",
				code: 4,
			},
		},
		{
			name:    "release unknown",
			release: fmt.Errorf("spooler: %w", spooler.ErrResultUnknown),
			exp: result{
				stderr: "write payload: context canceled; release: spooler:" +
					" result unknown; whether the message is released or still" +
					" leased is unknown (lease \"L1\")\n",
				code: 100,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(t.Context())
			defer cancel(nil)
			fake := &fakeClient{
				DoRecv: func(context.Context, spooler.RecvRequest) (spooler.RecvResult, bool, error) {
					// The signal lands with the payload on the wire: the
					// stream's next read fails with the cancellation.
					cancel(&cli.SignalError{
						Signal: syscall.SIGINT,
					})
					res := received()
					res.DataStream = io.NopCloser(iotest.ErrReader(context.Canceled))
					return res, true, nil
				},
				DoRelease: func(context.Context, spooler.ReleaseRequest) error {
					return test.release
				},
			}
			act := runCtx(t, ctx, fake, nil, "", "recv", "-s", "main", "-q", "jobs", "--ack")
			expect(t, act, test.exp)
		})
	}
}

// received is what the fake receive returns for a message: m1, leased as
// L1, with a five-byte payload.
func received() spooler.RecvResult {
	return spooler.RecvResult{
		ID:         "m1",
		CreatedAt:  time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC),
		Lease:      "L1",
		DataStream: io.NopCloser(strings.NewReader("hello")),
		DataSize:   5,
	}
}

// receivedOutput is received() as the CLI prints it and the lease file
// holds it.
func receivedOutput() recvResultOutput {
	return recvResultOutput{
		ID:        "m1",
		CreatedAt: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC),
		Lease:     "L1",
		DataSize:  5,
	}
}

func TestRecv(t *testing.T) {
	recvRecord := record(t, receivedOutput())
	for _, test := range []struct {
		name string
		args []string
		// ok is whether the fake receive finds a message.
		ok bool
		// expWait is the wait the fake must see, in seconds; nil means none.
		expWait *int
		// expAck is the lease the fake ack must see; empty means no ack.
		expAck string
		exp    result
	}{
		{
			name:   "ack",
			args:   []string{"recv", "-s", "main", "-q", "jobs", "--ack"},
			ok:     true,
			expAck: "L1",
			exp: result{
				stdout: "hello",
			},
		},
		{
			name: "nothing received",
			args: []string{"recv", "-s", "main", "-q", "jobs", "--ack"},
			exp: result{
				code: 1,
			},
		},
		{
			name: "include meta",
			args: []string{"recv", "-s", "main", "-q", "jobs", "-i"},
			ok:   true,
			exp: result{
				stdout: recvRecord + "\nhello",
			},
		},
		{
			name:   "include meta with ack",
			args:   []string{"recv", "-s", "main", "-q", "jobs", "-i", "--ack"},
			ok:     true,
			expAck: "L1",
			exp: result{
				stdout: recvRecord + "\nhello",
			},
		},
		{
			name:    "wait",
			args:    []string{"recv", "-s", "main", "-q", "jobs", "--wait", "20s", "--ack"},
			ok:      true,
			expWait: new(20),
			expAck:  "L1",
			exp: result{
				stdout: "hello",
			},
		},
		{
			name: "no lease disposition",
			args: []string{"recv", "-s", "main", "-q", "jobs"},
			exp: result{
				stderr: "a receive leases the message: pass --ack to settle it," +
					" --lease-file to keep the lease, or --include-meta to print it\n",
				code: 2,
			},
		},
		{
			name: "output format without include meta",
			args: []string{"recv", "-s", "main", "-q", "jobs", "--ack", "-o", "text"},
			exp: result{
				stderr: "--output takes effect only with --include-meta\n",
				code:   2,
			},
		},
		{
			// The output flags are checked before the receive: with the check
			// after it, a wrong combination would lease and release a message.
			name: "human with json",
			args: []string{"recv", "-s", "main", "-q", "jobs", "-i", "--human"},
			exp: result{
				stderr: "--human takes effect only with `-o table` or `-o text`\n",
				code:   2,
			},
		},
		{
			name: "human without include meta",
			args: []string{"recv", "-s", "main", "-q", "jobs", "--ack", "--human"},
			exp: result{
				stderr: "--human takes effect only with --include-meta\n",
				code:   2,
			},
		},
		{
			name: "compact without include meta",
			args: []string{"recv", "-s", "main", "-q", "jobs", "--ack", "--compact"},
			exp: result{
				stderr: "--compact takes effect only with --include-meta\n",
				code:   2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var (
				recvCalled bool
				wait       *int
				acked      string
			)
			fake := &fakeClient{
				DoRecv: func(_ context.Context, r spooler.RecvRequest) (spooler.RecvResult, bool, error) {
					recvCalled, wait = true, r.WaitSeconds
					if r.Spool != "main" || r.Queue != "jobs" {
						t.Errorf("request: %s/%s; want main/jobs", r.Spool, r.Queue)
					}
					if !r.Stream {
						t.Errorf("request does not stream the payload")
					}
					if !test.ok {
						return spooler.RecvResult{}, false, nil
					}
					return received(), true, nil
				},
				DoAck: func(_ context.Context, r spooler.AckRequest) error {
					acked = string(r.Lease)
					return nil
				},
			}
			act := run(t, fake, "", test.args...)
			expect(t, act, test.exp)
			if recvCalled != (test.exp.code != 2) {
				t.Errorf("recv called: %t", recvCalled)
			}
			if (wait == nil) != (test.expWait == nil) || wait != nil && *wait != *test.expWait {
				t.Errorf("wait: %v; want %v", wait, test.expWait)
			}
			if acked != test.expAck {
				t.Errorf("acked: %q; want %q", acked, test.expAck)
			}
		})
	}
}

func TestRecvFiles(t *testing.T) {
	dir := t.TempDir()
	leaseFile := filepath.Join(dir, "lease.json")
	dataFile := filepath.Join(dir, "payload.bin")
	fake := &fakeClient{
		DoRecv: func(context.Context, spooler.RecvRequest) (spooler.RecvResult, bool, error) {
			return received(), true, nil
		},
		DoAck: func(context.Context, spooler.AckRequest) error {
			return nil
		},
	}

	t.Run("lease file", func(t *testing.T) {
		act := run(t, fake, "", "recv", "-s", "main", "-q", "jobs", "-L", leaseFile)
		expect(t, act, result{
			stdout: "hello",
		})
		content, mode := readFile(t, leaseFile)
		exp := frames(t, receivedOutput())
		if content != exp {
			t.Errorf("lease file:\n%s\nwant:\n%s", content, exp)
		}
		if mode != 0600 {
			t.Errorf("lease file mode: %o; want 600", mode)
		}
	})

	t.Run("existing lease file is refused", func(t *testing.T) {
		act := run(t, fake, "", "recv", "-s", "main", "-q", "jobs", "-L", leaseFile)
		expect(t, act, result{
			stderr: "file exists: " + leaseFile + "; pass --overwrite to replace it\n",
			code:   2,
		})
	})

	t.Run("overwrite", func(t *testing.T) {
		act := run(t, fake, "", "recv", "-s", "main", "-q", "jobs", "-L", leaseFile, "--overwrite")
		expect(t, act, result{
			stdout: "hello",
		})
	})

	t.Run("overwrite tightens the mode", func(t *testing.T) {
		name := filepath.Join(dir, "shared.json")
		if err := os.WriteFile(name, []byte("old\n"), 0644); err != nil {
			t.Fatal(err)
		}
		act := run(t, fake, "", "recv", "-s", "main", "-q", "jobs", "-L", name, "--overwrite")
		expect(t, act, result{
			stdout: "hello",
		})
		content, mode := readFile(t, name)
		exp := frames(t, receivedOutput())
		if content != exp {
			t.Errorf("lease file:\n%s\nwant:\n%s", content, exp)
		}
		if mode != 0600 {
			t.Errorf("lease file mode: %o; want 600", mode)
		}
	})

	empty := &fakeClient{
		DoRecv: func(context.Context, spooler.RecvRequest) (spooler.RecvResult, bool, error) {
			return spooler.RecvResult{}, false, nil
		},
	}

	t.Run("nothing received leaves an existing file", func(t *testing.T) {
		name := filepath.Join(dir, "kept.json")
		if err := os.WriteFile(name, []byte("old\n"), 0644); err != nil {
			t.Fatal(err)
		}
		act := run(t, empty, "", "recv", "-s", "main", "-q", "jobs", "-L", name, "--overwrite")
		expect(t, act, result{
			code: 1,
		})
		content, mode := readFile(t, name)
		if content != "old\n" {
			t.Errorf("lease file: %q; want %q", content, "old\n")
		}
		if mode != 0644 {
			t.Errorf("lease file mode: %o; want 644", mode)
		}
	})

	t.Run("nothing received creates no file", func(t *testing.T) {
		name := filepath.Join(dir, "none.json")
		act := run(t, empty, "", "recv", "-s", "main", "-q", "jobs", "-L", name)
		expect(t, act, result{
			code: 1,
		})
		if _, err := os.Stat(name); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("lease file: %v; want not to exist", err)
		}
	})

	t.Run("failed ack keeps the lease file", func(t *testing.T) {
		name := filepath.Join(dir, "unacked.json")
		failing := &fakeClient{
			DoRecv: fake.DoRecv,
			DoAck: func(context.Context, spooler.AckRequest) error {
				return apiError(spooler.ErrorKindRateLimited, "slow down")
			},
		}
		act := run(t, failing, "", "recv", "-s", "main", "-q", "jobs", "-L", name, "--ack")
		expect(t, act, result{
			stdout: "hello",
			stderr: "ack: slow down (lease \"L1\")\n",
			code:   8,
		})
		content, _ := readFile(t, name)
		exp := frames(t, receivedOutput())
		if content != exp {
			t.Errorf("lease file:\n%s\nwant:\n%s", content, exp)
		}
	})

	t.Run("failed payload leaves an existing lease file", func(t *testing.T) {
		name := filepath.Join(dir, "previous.json")
		if err := os.WriteFile(name, []byte("old\n"), 0600); err != nil {
			t.Fatal(err)
		}
		var released string
		broken := &fakeClient{
			DoRecv: func(context.Context, spooler.RecvRequest) (spooler.RecvResult, bool, error) {
				res := received()
				res.DataStream = io.NopCloser(iotest.ErrReader(errors.New("boom")))
				return res, true, nil
			},
			DoRelease: func(_ context.Context, r spooler.ReleaseRequest) error {
				released = string(r.Lease)
				return nil
			},
		}
		act := run(t, broken, "", "recv", "-s", "main", "-q", "jobs", "-L", name, "--overwrite")
		expect(t, act, result{
			stderr: "write payload: boom; the lease was released\n",
			code:   4,
		})
		if released != "L1" {
			t.Errorf("released: %q; want %q", released, "L1")
		}
		content, _ := readFile(t, name)
		if content != "old\n" {
			t.Errorf("lease file: %q; want %q", content, "old\n")
		}
	})

	t.Run("failed release keeps the lease file", func(t *testing.T) {
		// The lease may be live, so the record is the one that matters: the
		// file the receive created stays, where a released lease leaves none.
		name := filepath.Join(dir, "unreleased.json")
		broken := &fakeClient{
			DoRecv: func(context.Context, spooler.RecvRequest) (spooler.RecvResult, bool, error) {
				res := received()
				res.DataStream = io.NopCloser(iotest.ErrReader(errors.New("boom")))
				return res, true, nil
			},
			DoRelease: func(context.Context, spooler.ReleaseRequest) error {
				return apiError(spooler.ErrorKindRateLimited, "slow down")
			},
		}
		act := run(t, broken, "", "recv", "-s", "main", "-q", "jobs", "-L", name)
		expect(t, act, result{
			stderr: "write payload: boom; release: slow down (lease \"L1\")\n",
			code:   4,
		})
		content, _ := readFile(t, name)
		exp := frames(t, receivedOutput())
		if content != exp {
			t.Errorf("lease file:\n%s\nwant:\n%s", content, exp)
		}
	})

	t.Run("unknown release replaces the lease file", func(t *testing.T) {
		// The lease may be live, so the new record lands over the old one,
		// and the exit says the one thing in doubt is the release.
		name := filepath.Join(dir, "doubtful.json")
		if err := os.WriteFile(name, []byte("old\n"), 0600); err != nil {
			t.Fatal(err)
		}
		broken := &fakeClient{
			DoRecv: func(context.Context, spooler.RecvRequest) (spooler.RecvResult, bool, error) {
				res := received()
				res.DataStream = io.NopCloser(iotest.ErrReader(errors.New("boom")))
				return res, true, nil
			},
			DoRelease: func(context.Context, spooler.ReleaseRequest) error {
				return fmt.Errorf("spooler: %w", spooler.ErrResultUnknown)
			},
		}
		act := run(t, broken, "", "recv", "-s", "main", "-q", "jobs", "-L", name, "--overwrite")
		expect(t, act, result{
			stderr: "write payload: boom; release: spooler: result unknown;" +
				" whether the message is released or still leased is unknown" +
				" (lease \"L1\")\n",
			code: 100,
		})
		content, _ := readFile(t, name)
		exp := frames(t, receivedOutput())
		if content != exp {
			t.Errorf("lease file:\n%s\nwant:\n%s", content, exp)
		}
	})

	t.Run("aliased lease and data file", func(t *testing.T) {
		// One file under two spellings is one destination, with the record
		// ahead of the payload, not a payload renamed over the record.
		if err := os.Mkdir(filepath.Join(dir, "real"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("real", filepath.Join(dir, "alias")); err != nil {
			t.Fatal(err)
		}
		name := filepath.Join(dir, "real", "both.bin")
		other := filepath.Join(dir, "alias", "both.bin")
		act := run(t, fake, "", "recv", "-s", "main", "-q", "jobs", "-L", name, "-D", other, "--overwrite")
		expect(t, act, result{})
		content, mode := readFile(t, name)
		exp := frames(t, receivedOutput()) + "\nhello"
		if content != exp {
			t.Errorf("file:\n%s\nwant:\n%s", content, exp)
		}
		if mode != 0600 {
			t.Errorf("file mode: %o; want 600", mode)
		}
	})

	t.Run("lease file is the stdout redirect", func(t *testing.T) {
		// run points stdout at .stdout in the working directory, as a shell
		// redirect does: the file is stdout, not one to replace, since a
		// replacement would unlink the payload stdout wrote.
		act := run(t, fake, "", "recv", "-s", "main", "-q", "jobs", "-L", ".stdout", "--ack", "--overwrite")
		expect(t, act, result{
			stdout: frames(t, receivedOutput()) + "\nhello",
		})
	})

	t.Run("data file is the stdout redirect", func(t *testing.T) {
		act := run(t, fake, "", "recv", "-s", "main", "-q", "jobs", "-i", "-D", ".stdout")
		expect(t, act, result{
			stdout: record(t, receivedOutput()) + "\nhello",
		})
	})

	t.Run("file symlink names its target", func(t *testing.T) {
		// The replacement lands on the file the link points at, and the
		// link stays a link.
		target := filepath.Join(dir, "target.bin")
		link := filepath.Join(dir, "target-link")
		if err := os.WriteFile(target, []byte("old\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("target.bin", link); err != nil {
			t.Fatal(err)
		}
		act := run(t, fake, "", "recv", "-s", "main", "-q", "jobs", "-L", link, "-D", target, "--overwrite")
		expect(t, act, result{})
		content, mode := readFile(t, target)
		exp := frames(t, receivedOutput()) + "\nhello"
		if content != exp {
			t.Errorf("target:\n%s\nwant:\n%s", content, exp)
		}
		if mode != 0600 {
			t.Errorf("target mode: %o; want 600", mode)
		}
		if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("link: %v, mode %v; want a symlink", err, info.Mode())
		}
	})

	t.Run("hard links are refused", func(t *testing.T) {
		one := filepath.Join(dir, "one.bin")
		two := filepath.Join(dir, "two.bin")
		if err := os.WriteFile(one, []byte("old\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(one, two); err != nil {
			t.Fatal(err)
		}
		act := run(t, fake, "", "recv", "-s", "main", "-q", "jobs", "-L", one, "--overwrite")
		expect(t, act, result{
			stderr: one + " has other names (hard links): a replacement would" +
				" update only this one\n",
			code: 2,
		})
		content, _ := readFile(t, one)
		if content != "old\n" {
			t.Errorf("file: %q; want %q", content, "old\n")
		}
	})

	t.Run("spellings that differ in case", func(t *testing.T) {
		// On a case-insensitive filesystem two spellings are one file that
		// no resolution makes equal; identity does.
		lower := filepath.Join(dir, "mixed.bin")
		upper := filepath.Join(dir, "MIXED.bin")
		if err := os.WriteFile(lower, []byte("old\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(upper); err != nil {
			t.Skip("case-sensitive filesystem")
		}
		act := run(t, fake, "", "recv", "-s", "main", "-q", "jobs", "-L", lower, "-D", upper, "--overwrite")
		expect(t, act, result{})
		content, _ := readFile(t, lower)
		exp := frames(t, receivedOutput()) + "\nhello"
		if content != exp {
			t.Errorf("file:\n%s\nwant:\n%s", content, exp)
		}
	})

	t.Run("dot-dot through a directory symlink", func(t *testing.T) {
		// The OS follows the link before it climbs, so view/../bundle with
		// view -> actual/child is actual/bundle, not bundle beside view.
		if err := os.MkdirAll(filepath.Join(dir, "actual", "child"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join("actual", "child"), filepath.Join(dir, "view")); err != nil {
			t.Fatal(err)
		}
		beside := filepath.Join(dir, "bundle")
		if err := os.WriteFile(beside, []byte("old\n"), 0644); err != nil {
			t.Fatal(err)
		}
		// Not filepath.Join, which would collapse the ".." here.
		name := filepath.Join(dir, "view") + "/../bundle"
		act := run(t, fake, "", "recv", "-s", "main", "-q", "jobs", "-L", name, "--ack")
		expect(t, act, result{
			stdout: "hello",
		})
		content, _ := readFile(t, filepath.Join(dir, "actual", "bundle"))
		exp := frames(t, receivedOutput())
		if content != exp {
			t.Errorf("file:\n%s\nwant:\n%s", content, exp)
		}
		if content, _ := readFile(t, beside); content != "old\n" {
			t.Errorf("file beside the link: %q; want %q", content, "old\n")
		}
	})

	t.Run("unknown ack keeps the lease file", func(t *testing.T) {
		name := filepath.Join(dir, "unknown.json")
		unknown := &fakeClient{
			DoRecv: fake.DoRecv,
			DoAck: func(context.Context, spooler.AckRequest) error {
				return fmt.Errorf("spooler: %w", spooler.ErrResultUnknown)
			},
		}
		act := run(t, unknown, "", "recv", "-s", "main", "-q", "jobs", "-L", name, "--ack")
		expect(t, act, result{
			stdout: "hello",
			stderr: "ack: spooler: result unknown; whether the message is settled" +
				" or still leased is unknown (lease \"L1\")\n",
			code: 100,
		})
		if _, err := os.Stat(name); err != nil {
			t.Errorf("lease file: %v", err)
		}
	})

	t.Run("data file", func(t *testing.T) {
		act := run(t, fake, "", "recv", "-s", "main", "-q", "jobs", "--ack", "-D", dataFile)
		if _, err := os.Stat(dataFile); err != nil {
			t.Fatalf("data file: %v", err)
		}
		expect(t, act, result{})
		content, _ := readFile(t, dataFile)
		if content != "hello" {
			t.Errorf("data file: %q; want %q", content, "hello")
		}
	})

	t.Run("no temporary file remains", func(t *testing.T) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") {
				t.Errorf("left behind: %s", e.Name())
			}
		}
	})
}
