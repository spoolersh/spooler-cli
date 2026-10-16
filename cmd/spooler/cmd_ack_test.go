package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spoolersh/spooler-go"
)

func TestAck(t *testing.T) {
	dir := t.TempDir()
	// The file is what recv --lease-file --data-file writes to one file: the
	// record, then the payload.
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
		// err is what the fake ack returns.
		err error
		// expLease is the lease the fake must see; empty means no call.
		expLease string
		exp      result
	}{
		{
			name:     "lease flag",
			args:     []string{"ack", "-s", "main", "-l", "L1"},
			expLease: "L1",
		},
		{
			name:     "lease file with the payload after the record",
			args:     []string{"ack", "-s", "main", "-L", leaseFile},
			expLease: "L2",
		},
		{
			name: "no lease",
			args: []string{"ack", "-s", "main"},
			exp: result{
				stderr: "no lease: pass --lease or --lease-file\n",
				code:   2,
			},
		},
		{
			name: "argument",
			args: []string{"ack", "-s", "main", "L1"},
			exp: result{
				stderr: "unexpected argument: \"L1\"\n",
				code:   2,
			},
		},
		{
			name: "both",
			args: []string{"ack", "-s", "main", "-l", "L1", "-L", leaseFile},
			exp: result{
				stderr: "--lease and --lease-file are mutually exclusive\n",
				code:   2,
			},
		},
		{
			name: "missing lease file",
			args: []string{"ack", "-s", "main", "-L", filepath.Join(dir, "none.json")},
			exp: result{
				stderr: "open " + filepath.Join(dir, "none.json") + ": no such file or directory\n",
				code:   4,
			},
		},
		{
			name:     "stale lease",
			args:     []string{"ack", "-s", "main", "-l", "L1"},
			err:      apiError(spooler.ErrorKindStaleToken, "lease is stale"),
			expLease: "L1",
			exp: result{
				stderr: "lease is stale\n",
				code:   4,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var lease string
			fake := &fakeClient{
				DoAck: func(_ context.Context, r spooler.AckRequest) error {
					lease = string(r.Lease)
					if r.Spool != "main" {
						t.Errorf("spool: %q; want %q", r.Spool, "main")
					}
					return test.err
				},
			}
			act := run(t, fake, "", test.args...)
			expect(t, act, test.exp)
			if lease != test.expLease {
				t.Errorf("lease: %q; want %q", lease, test.expLease)
			}
		})
	}
}
