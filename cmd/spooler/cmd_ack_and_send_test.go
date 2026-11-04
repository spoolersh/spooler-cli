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

func TestAckAndSend(t *testing.T) {
	dir := t.TempDir()
	leaseFile := filepath.Join(dir, "lease.json")
	leased := frames(t, recvResultOutput{
		ID:    "m1",
		Lease: "L2",
	})
	if err := os.WriteFile(leaseFile, []byte(leased+"\nhello"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		args []string
		// res and err are what the fake ack-and-send returns.
		res spooler.AckAndSendResult
		err error
		// expReq is the request the fake must see, its DataSource aside;
		// nil means ack-and-send must not be called.
		expReq *spooler.AckAndSendRequest
		exp    result
	}{
		{
			name: "lease flag",
			args: []string{"ack-and-send", "-s", "main", "-l", "L1", "-q", "next", "-d", "hello"},
			res: spooler.AckAndSendResult{
				ID: "m2",
			},
			expReq: &spooler.AckAndSendRequest{
				Spool: "main",
				Lease: "L1",
				Queue: "next",
			},
			exp: result{
				stdout: record(t, ackAndSendResultOutput{
					ID: "m2",
				}),
			},
		},
		{
			name: "lease file with the payload after the record",
			args: []string{"ack-and-send", "-s", "main", "-L", leaseFile, "-q", "next", "-d", "hello"},
			res: spooler.AckAndSendResult{
				ID: "m2",
			},
			expReq: &spooler.AckAndSendRequest{
				Spool: "main",
				Lease: "L2",
				Queue: "next",
			},
			exp: result{
				stdout: record(t, ackAndSendResultOutput{
					ID: "m2",
				}),
			},
		},
		{
			name: "delay, dedup and handle",
			args: []string{"ack-and-send", "-s", "main", "-l", "L1", "-q", "next", "-d", "hello", "--delay", "30s", "--dedup-string", "k1", "--return-handle"},
			res: spooler.AckAndSendResult{
				ID:     "m2",
				Handle: "H2",
			},
			expReq: &spooler.AckAndSendRequest{
				Spool:        "main",
				Lease:        "L1",
				Queue:        "next",
				DelaySeconds: 30,
				Dedup:        spooler.DedupString("k1"),
				ReturnHandle: true,
			},
			exp: result{
				stdout: record(t, ackAndSendResultOutput{
					ID:     "m2",
					Handle: "H2",
				}),
			},
		},
		{
			name: "no lease",
			args: []string{"ack-and-send", "-s", "main", "-q", "next", "-d", "hello"},
			exp: result{
				stderr: "no lease: pass --lease or --lease-file\n",
				code:   2,
			},
		},
		{
			name: "both leases",
			args: []string{"ack-and-send", "-s", "main", "-l", "L1", "-L", leaseFile, "-q", "next", "-d", "hello"},
			exp: result{
				stderr: "--lease and --lease-file are mutually exclusive\n",
				code:   2,
			},
		},
		{
			name: "no payload",
			args: []string{"ack-and-send", "-s", "main", "-l", "L1", "-q", "next"},
			exp: result{
				stderr: "no payload: pass --data, --data-file, or `--data-file -` for stdin\n",
				code:   2,
			},
		},
		{
			name: "both payloads",
			args: []string{"ack-and-send", "-s", "main", "-l", "L1", "-q", "next", "-d", "x", "-D", "payload.bin"},
			exp: result{
				stderr: "--data and --data-file are mutually exclusive\n",
				code:   2,
			},
		},
		{
			name: "both dedup keys",
			args: []string{"ack-and-send", "-s", "main", "-l", "L1", "-q", "next", "-d", "x", "--dedup-string", "k1", "--dedup-hash", "ab"},
			exp: result{
				stderr: "--dedup-hash and --dedup-string are mutually exclusive\n",
				code:   2,
			},
		},
		{
			name: "argument hides flags",
			args: []string{"ack-and-send", "-s", "main", "-l", "L1", "payload", "-q", "next", "-d", "x"},
			exp: result{
				stderr: "unexpected argument: \"payload\"\n",
				code:   2,
			},
		},
		{
			name: "dedup claimed acks nothing",
			args: []string{"ack-and-send", "-s", "main", "-l", "L1", "-q", "next", "-d", "x", "--dedup-string", "k1"},
			err:  apiError(spooler.ErrorKindDedupClaimed, "dedup key claimed"),
			expReq: &spooler.AckAndSendRequest{
				Spool: "main",
				Lease: "L1",
				Queue: "next",
				Dedup: spooler.DedupString("k1"),
			},
			exp: result{
				stderr: "dedup key claimed\n",
				code:   4,
			},
		},
		{
			name: "stale lease",
			args: []string{"ack-and-send", "-s", "main", "-l", "L1", "-q", "next", "-d", "x"},
			err:  apiError(spooler.ErrorKindStaleToken, "lease is stale"),
			expReq: &spooler.AckAndSendRequest{
				Spool: "main",
				Lease: "L1",
				Queue: "next",
			},
			exp: result{
				stderr: "lease is stale\n",
				code:   4,
			},
		},
		{
			name: "rate limited",
			args: []string{"ack-and-send", "-s", "main", "-l", "L1", "-q", "next", "-d", "x"},
			err:  apiError(spooler.ErrorKindRateLimited, "slow down"),
			expReq: &spooler.AckAndSendRequest{
				Spool: "main",
				Lease: "L1",
				Queue: "next",
			},
			exp: result{
				stderr: "slow down\n",
				code:   8,
			},
		},
		{
			name: "result unknown",
			args: []string{"ack-and-send", "-s", "main", "-l", "L1", "-q", "next", "-d", "x"},
			err:  fmt.Errorf("spooler: %w", spooler.ErrResultUnknown),
			expReq: &spooler.AckAndSendRequest{
				Spool: "main",
				Lease: "L1",
				Queue: "next",
			},
			exp: result{
				stderr: "spooler: result unknown; a retry is safe: a retry of a call" +
					" that committed is refused as stale and sends nothing\n",
				code: 100,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var (
				called bool
				req    spooler.AckAndSendRequest
			)
			fake := &fakeClient{
				DoAckAndSend: func(_ context.Context, r spooler.AckAndSendRequest) (spooler.AckAndSendResult, error) {
					called, req = true, r
					return test.res, test.err
				},
			}
			act := run(t, fake, "", test.args...)
			expect(t, act, test.exp)
			if called != (test.expReq != nil) {
				t.Fatalf("ack-and-send called: %t; want %t", called, test.expReq != nil)
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
