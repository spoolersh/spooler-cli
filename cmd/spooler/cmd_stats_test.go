package main

import (
	"context"
	"testing"

	"github.com/spoolersh/spooler-go"
)

func TestStats(t *testing.T) {
	spool := spooler.Spool{
		Name:       "main",
		QueueCount: 2,
		Stats: spooler.MessageStats{
			Total:   5,
			Visible: 3,
			Pending: 2,
		},
	}
	described := spoolOutput{
		Name:       "main",
		QueueCount: 2,
		Stats: messageStatsOutput{
			Total:   5,
			Visible: 3,
			Pending: 2,
		},
	}
	for _, test := range []struct {
		name string
		args []string
		exp  result
	}{
		{
			name: "record",
			args: []string{"stats", "-s", "main"},
			exp: result{
				stdout: record(t, described),
			},
		},
		{
			name: "argument",
			args: []string{"stats", "main"},
			exp: result{
				stderr: "unexpected argument: \"main\"\n",
				code:   2,
			},
		},
		{
			name: "compact",
			args: []string{"stats", "-s", "main", "--compact"},
			exp: result{
				stdout: frames(t, described),
			},
		},
		{
			name: "template",
			args: []string{"stats", "-s", "main", "--template", "{{.name}} has {{.stats.visible}} visible"},
			exp: result{
				stdout: "main has 3 visible\n",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeClient{
				DoSpool: func(_ context.Context, r spooler.SpoolRequest) (spooler.Spool, error) {
					if r.Spool != "main" {
						t.Errorf("spool: %q; want %q", r.Spool, "main")
					}
					return spool, nil
				},
			}
			act := run(t, fake, "", test.args...)
			expect(t, act, test.exp)
		})
	}
}
