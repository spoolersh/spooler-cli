package main

import (
	"context"
	"testing"

	"github.com/spoolersh/spooler-go"
)

func TestNack(t *testing.T) {
	for _, test := range []struct {
		name     string
		args     []string
		err      error
		expLease string
		exp      result
	}{
		{
			name:     "lease flag",
			args:     []string{"nack", "-s", "main", "-l", "L1"},
			expLease: "L1",
		},
		{
			name: "no lease",
			args: []string{"nack", "-s", "main"},
			exp: result{
				stderr: "no lease: pass --lease or --lease-file\n",
				code:   2,
			},
		},
		{
			name:     "queue busy is retry later",
			args:     []string{"nack", "-s", "main", "-l", "L1"},
			err:      apiError(spooler.ErrorKindQueueBusy, "queue is busy"),
			expLease: "L1",
			exp: result{
				stderr: "queue is busy\n",
				code:   8,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var lease string
			fake := &fakeClient{
				DoNack: func(_ context.Context, r spooler.NackRequest) error {
					lease = string(r.Lease)
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
