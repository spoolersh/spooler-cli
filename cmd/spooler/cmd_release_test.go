package main

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spoolersh/spooler-go"
)

func TestRelease(t *testing.T) {
	for _, test := range []struct {
		name   string
		args   []string
		expReq *spooler.ReleaseRequest
		exp    result
	}{
		{
			name: "at once",
			args: []string{"release", "-s", "main", "-l", "L1"},
			expReq: &spooler.ReleaseRequest{
				Spool: "main",
				Lease: "L1",
			},
		},
		{
			name: "with a delay",
			args: []string{"release", "-s", "main", "-l", "L1", "--delay", "5m"},
			expReq: &spooler.ReleaseRequest{
				Spool:        "main",
				Lease:        "L1",
				DelaySeconds: 300,
			},
		},
		{
			name: "bare delay is seconds",
			args: []string{"release", "-s", "main", "-l", "L1", "--delay", "30"},
			expReq: &spooler.ReleaseRequest{
				Spool:        "main",
				Lease:        "L1",
				DelaySeconds: 30,
			},
		},
		{
			name: "fractional delay",
			args: []string{"release", "-s", "main", "-l", "L1", "--delay", "1.5s"},
			exp: result{
				stderr: "--delay: duration \"1.5s\": want whole seconds, as 90 or 1m30s\n",
				code:   2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var req *spooler.ReleaseRequest
			fake := &fakeClient{
				DoRelease: func(_ context.Context, r spooler.ReleaseRequest) error {
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
