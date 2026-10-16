package main

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spoolersh/spooler-go"
)

func TestRecover(t *testing.T) {
	for _, test := range []struct {
		name   string
		args   []string
		err    error
		expReq *spooler.RecoverRequest
		exp    result
	}{
		{
			name: "at once",
			args: []string{"recover", "-s", "main", "--handle", "H1"},
			expReq: &spooler.RecoverRequest{
				Spool:  "main",
				Handle: "H1",
			},
		},
		{
			name: "with a delay",
			args: []string{"recover", "-s", "main", "--handle", "H1", "--delay", "1h"},
			expReq: &spooler.RecoverRequest{
				Spool:        "main",
				Handle:       "H1",
				DelaySeconds: 3600,
			},
		},
		{
			name: "no handle",
			args: []string{"recover", "-s", "main"},
			exp: result{
				stderr: "no message: pass --handle\n",
				code:   2,
			},
		},
		{
			name: "not failed",
			args: []string{"recover", "-s", "main", "--handle", "H1"},
			err:  apiError(spooler.ErrorKindNotFailed, "message m1 is not failed"),
			expReq: &spooler.RecoverRequest{
				Spool:  "main",
				Handle: "H1",
			},
			exp: result{
				stderr: "message m1 is not failed\n",
				code:   4,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var req *spooler.RecoverRequest
			fake := &fakeClient{
				DoRecover: func(_ context.Context, r spooler.RecoverRequest) error {
					req = &r
					return test.err
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
