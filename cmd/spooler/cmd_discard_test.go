package main

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spoolersh/spooler-go"
)

func TestDiscard(t *testing.T) {
	for _, test := range []struct {
		name   string
		args   []string
		expReq *spooler.DiscardRequest
		exp    result
	}{
		{
			name: "by handle",
			args: []string{"discard", "-s", "main", "--handle", "H1", "--yes"},
			expReq: &spooler.DiscardRequest{
				Spool:  "main",
				Handle: "H1",
			},
		},
		{
			name: "by lease",
			args: []string{"discard", "-s", "main", "-l", "L1", "--yes"},
			expReq: &spooler.DiscardRequest{
				Spool: "main",
				Lease: "L1",
			},
		},
		{
			name: "without consent",
			args: []string{"discard", "-s", "main", "--handle", "H1"},
			exp: result{
				stderr: "discard drops the message and its payload for good: pass --yes to confirm\n",
				code:   2,
			},
		},
		{
			name: "neither handle nor lease",
			args: []string{"discard", "-s", "main", "--yes"},
			exp: result{
				stderr: "no message: pass --handle, --lease or --lease-file\n",
				code:   2,
			},
		},
		{
			name: "handle and lease",
			args: []string{"discard", "-s", "main", "--handle", "H1", "-l", "L1", "--yes"},
			exp: result{
				stderr: "--handle and --lease or --lease-file are mutually exclusive\n",
				code:   2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var req *spooler.DiscardRequest
			fake := &fakeClient{
				DoDiscard: func(_ context.Context, r spooler.DiscardRequest) error {
					req = &r
					return nil
				},
			}
			act := run(t, fake, "", test.args...)
			expect(t, act, test.exp)
			if diff := cmp.Diff(test.expReq, req); diff != "" {
				t.Errorf("request:\n%s", diff)
			}
		})
	}
}
