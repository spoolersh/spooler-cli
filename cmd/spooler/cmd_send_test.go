package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spoolersh/spooler-go"
)

func TestSend(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		// res and err are what the fake send returns.
		res spooler.SendResult
		err error
		// expReq is the request the fake must see, its DataSource aside;
		// nil means send must not be called.
		expReq *spooler.SendRequest
		exp    result
	}{
		{
			name: "data flag",
			args: []string{"send", "-s", "main", "-q", "jobs", "-d", "hello"},
			res: spooler.SendResult{
				ID: "m1",
			},
			expReq: &spooler.SendRequest{
				Spool: "main",
				Queue: "jobs",
			},
			exp: result{
				stdout: record(t, sendResultOutput{
					ID: "m1",
				}),
			},
		},
		{
			name: "delay and dedup",
			args: []string{"send", "-s", "main", "-q", "jobs", "-d", "hello", "--delay", "30s", "--dedup-string", "k1"},
			res: spooler.SendResult{
				ID:        "m1",
				Duplicate: true,
			},
			expReq: &spooler.SendRequest{
				Spool:        "main",
				Queue:        "jobs",
				DelaySeconds: 30,
				Dedup:        spooler.DedupString("k1"),
			},
			exp: result{
				stdout: record(t, sendResultOutput{
					ID:        "m1",
					Duplicate: true,
				}),
			},
		},
		{
			name: "return handle",
			args: []string{"send", "-s", "main", "-q", "jobs", "-d", "hello", "--return-handle"},
			res: spooler.SendResult{
				ID:     "m1",
				Handle: "H1",
			},
			expReq: &spooler.SendRequest{
				Spool:        "main",
				Queue:        "jobs",
				ReturnHandle: true,
			},
			exp: result{
				stdout: record(t, sendResultOutput{
					ID:     "m1",
					Handle: "H1",
				}),
			},
		},
		{
			name: "no payload",
			args: []string{"send", "-s", "main", "-q", "jobs"},
			exp: result{
				stderr: "no payload: pass --data, --data-file, or `--data-file -` for stdin\n",
				code:   2,
			},
		},
		{
			name: "both payloads",
			args: []string{"send", "-s", "main", "-q", "jobs", "-d", "x", "-D", "payload.bin"},
			exp: result{
				stderr: "--data and --data-file are mutually exclusive\n",
				code:   2,
			},
		},
		{
			name: "empty dedup string goes to the api",
			args: []string{"send", "-s", "main", "-q", "jobs", "-d", "x", "--dedup-string", ""},
			res: spooler.SendResult{
				ID: "m1",
			},
			expReq: &spooler.SendRequest{
				Spool: "main",
				Queue: "jobs",
				Dedup: spooler.DedupString(""),
			},
			exp: result{
				stdout: record(t, sendResultOutput{
					ID: "m1",
				}),
			},
		},
		{
			name: "empty dedup hash goes to the api",
			args: []string{"send", "-s", "main", "-q", "jobs", "-d", "x", "--dedup-hash", ""},
			res: spooler.SendResult{
				ID: "m1",
			},
			expReq: &spooler.SendRequest{
				Spool: "main",
				Queue: "jobs",
				Dedup: spooler.DedupHash(""),
			},
			exp: result{
				stdout: record(t, sendResultOutput{
					ID: "m1",
				}),
			},
		},
		{
			// The parser stops at the first word that is not a flag, so the
			// flags behind it were never read.
			name: "argument hides flags",
			args: []string{"send", "-s", "main", "-q", "jobs", "-d", "x", "payload", "--dedup-string", "k1"},
			exp: result{
				stderr: "unexpected argument: \"payload\"\n",
				code:   2,
			},
		},
		{
			name: "rate limited",
			args: []string{"send", "-s", "main", "-q", "jobs", "-d", "hello"},
			err:  apiError(spooler.ErrorKindRateLimited, "slow down"),
			expReq: &spooler.SendRequest{
				Spool: "main",
				Queue: "jobs",
			},
			exp: result{
				stderr: "slow down\n",
				code:   8,
			},
		},
		{
			name: "result unknown without a dedup key",
			args: []string{"send", "-s", "main", "-q", "jobs", "-d", "hello"},
			err:  fmt.Errorf("spooler: %w", spooler.ErrResultUnknown),
			expReq: &spooler.SendRequest{
				Spool: "main",
				Queue: "jobs",
			},
			exp: result{
				stderr: "spooler: result unknown; the message may have been sent," +
					" and a retry may send it twice: only a send whose first" +
					" attempt carried --dedup-string or --dedup-hash is safe to" +
					" retry, with the same value, within the queue's dedup window\n",
				code: 100,
			},
		},
		{
			name: "result unknown with a dedup key",
			args: []string{"send", "-s", "main", "-q", "jobs", "-d", "hello", "--dedup-string", "k1"},
			err:  fmt.Errorf("spooler: %w", spooler.ErrResultUnknown),
			expReq: &spooler.SendRequest{
				Spool: "main",
				Queue: "jobs",
				Dedup: spooler.DedupString("k1"),
			},
			exp: result{
				stderr: "spooler: result unknown; a retry with the same --dedup-string" +
					" or --dedup-hash is safe within the queue's dedup window\n",
				code: 100,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var (
				called bool
				req    spooler.SendRequest
			)
			fake := &fakeClient{
				DoSend: func(_ context.Context, r spooler.SendRequest) (spooler.SendResult, error) {
					called, req = true, r
					return test.res, test.err
				},
			}
			act := run(t, fake, "", test.args...)
			expect(t, act, test.exp)
			if called != (test.expReq != nil) {
				t.Fatalf("send called: %t; want %t", called, test.expReq != nil)
			}
			if test.expReq == nil {
				return
			}
			if req.DataSource.IsZero() {
				t.Errorf("request has no data source")
			}
			req.DataSource = spooler.DataSource{}
			if diff := cmp.Diff(*test.expReq, req, cmpopts.EquateComparable(spooler.Dedup{}, spooler.DataSource{})); diff != "" {
				t.Errorf("request:\n%s", diff)
			}
		})
	}
}

func TestSendDataFile(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "payload.bin")
	if err := os.WriteFile(name, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	var called bool
	fake := &fakeClient{
		DoSend: func(_ context.Context, r spooler.SendRequest) (spooler.SendResult, error) {
			called = true
			if r.DataSource.IsZero() {
				t.Errorf("request has no data source")
			}
			return spooler.SendResult{
				ID: "m1",
			}, nil
		},
	}
	act := run(t, fake, "", "send", "-s", "main", "-q", "jobs", "-D", name)
	expect(t, act, result{
		stdout: record(t, sendResultOutput{
			ID: "m1",
		}),
	})
	if !called {
		t.Errorf("send not called")
	}
}
