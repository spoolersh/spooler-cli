package main

import (
	"context"
	"testing"
	"time"

	"github.com/spoolersh/spooler-go"
)

func TestRenew(t *testing.T) {
	expires := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name     string
		args     []string
		res      spooler.RenewResult
		expLease string
		exp      result
	}{
		{
			name: "new expiry",
			args: []string{"renew", "-s", "main", "-l", "L1"},
			res: spooler.RenewResult{
				LeaseExpiresAt: expires,
			},
			expLease: "L1",
			exp: result{
				stdout: record(t, renewResultOutput{
					LeaseExpiresAt: expires,
				}),
			},
		},
		{
			name:     "no expiry",
			args:     []string{"renew", "-s", "main", "-l", "L1"},
			expLease: "L1",
			exp: result{
				stdout: record(t, renewResultOutput{}),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var lease string
			fake := &fakeClient{
				DoRenew: func(_ context.Context, r spooler.RenewRequest) (spooler.RenewResult, error) {
					lease = string(r.Lease)
					return test.res, nil
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
