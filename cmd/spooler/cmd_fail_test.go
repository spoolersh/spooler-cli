package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spoolersh/spooler-go"
)

func TestFail(t *testing.T) {
	dir := t.TempDir()
	leaseFile := filepath.Join(dir, "lease.json")
	leased := frames(t, recvResultOutput{
		ID:    "m1",
		Lease: "L2",
	})
	if err := os.WriteFile(leaseFile, []byte(leased), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		args     []string
		expLease string
		exp      result
	}{
		{
			name:     "lease flag",
			args:     []string{"fail", "-s", "main", "-l", "L1"},
			expLease: "L1",
		},
		{
			name:     "lease file",
			args:     []string{"fail", "-s", "main", "-L", leaseFile},
			expLease: "L2",
		},
		{
			name: "lease file without a lease",
			args: []string{"fail", "-s", "main", "-L", "-"},
			exp: result{
				stderr: "no lease in -\n",
				code:   2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var lease string
			fake := &fakeClient{
				DoFail: func(_ context.Context, r spooler.FailRequest) error {
					lease = string(r.Lease)
					return nil
				},
			}
			// Stdin holds a record without a lease, for the "-" case.
			noLease := frames(t, recvResultOutput{
				ID: "m1",
			})
			act := run(t, fake, noLease, test.args...)
			expect(t, act, test.exp)
			if lease != test.expLease {
				t.Errorf("lease: %q; want %q", lease, test.expLease)
			}
		})
	}
}
