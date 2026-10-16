package main

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/spoolersh/spooler-go"
)

// TestQueuesListPageFailure pins that a listing cut by a failed page prints
// the queues fetched before it whole, in the buffering formats too, then
// the error.
func TestQueuesListPageFailure(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		exp  result
	}{
		{
			name: "csv",
			args: []string{"queues", "list", "-s", "main", "-o", "csv", "-c", "name,state"},
			exp: result{
				stdout: "name,state\njobs,active\n",
				stderr: "spooler: unavailable\n",
				code:   8,
			},
		},
		{
			name: "table",
			args: []string{"queues", "list", "-s", "main", "-o", "table", "-c", "name,state"},
			exp: result{
				stdout: "NAME  STATE\njobs  active\n",
				stderr: "spooler: unavailable\n",
				code:   8,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeClient{
				DoQueues: func(_ context.Context, r spooler.QueuesRequest) (spooler.Page[spooler.Queue], error) {
					if r.After != "" {
						return spooler.Page[spooler.Queue]{}, fmt.Errorf(
							"spooler: %w",
							spooler.ErrUnavailable,
						)
					}
					return spooler.Page[spooler.Queue]{
						Items: []spooler.Queue{
							{
								Name:  "jobs",
								State: spooler.QueueStateActive,
							},
						},
						Next: "c1",
					}, nil
				},
			}
			expect(t, run(t, fake, "", test.args...), test.exp)
		})
	}
}

func TestQueuesList(t *testing.T) {
	queues := []spooler.Queue{
		{
			Name:  "jobs",
			State: spooler.QueueStateActive,
		},
		{
			Name:  "orders",
			State: spooler.QueueStateDraining,
		},
	}
	jobs := queueOutput{
		Name:  "jobs",
		State: "active",
	}
	orders := queueOutput{
		Name:  "orders",
		State: "draining",
	}
	for _, test := range []struct {
		name     string
		args     []string
		expMatch string
		exp      result
	}{
		{
			name: "every queue",
			args: []string{"queues", "list", "-s", "main", "-o", "json", "--compact"},
			exp: result{
				stdout: frames(t, jobs, orders),
			},
		},
		{
			name: "alias",
			args: []string{"queues", "ls", "-s", "main", "-o", "json", "--compact"},
			exp: result{
				stdout: frames(t, jobs, orders),
			},
		},
		{
			name: "limit",
			args: []string{"queues", "list", "-s", "main", "-o", "json", "--compact", "-n", "1"},
			exp: result{
				stdout: frames(t, jobs),
			},
		},
		{
			name:     "match is passed on",
			args:     []string{"queues", "list", "-s", "main", "-o", "json", "--compact", "--match", "^jo"},
			expMatch: "^jo",
			exp: result{
				stdout: frames(t, jobs, orders),
			},
		},
		{
			name: "template with an unknown field",
			args: []string{"queues", "list", "-s", "main", "--template", "{{.created_af}}"},
			exp: result{
				stderr: "template: --template:1:2: executing \"--template\" at <.created_af>: map has no entry for key \"created_af\"; fields: name, uid, createdAt, state, settings.leaseTimeoutSeconds, settings.maxRetries, settings.dedupWindowSeconds, settings.retentionSeconds, settings.effectiveRetentionSeconds, settings.recvRateLimit.intervalMicros, settings.recvRateLimit.burst, stats.sampledAt, stats.startedAt, stats.total, stats.dataBytes, stats.delayed, stats.visible, stats.pending, stats.failed, stats.unavailable, stats.addedTotal, stats.ackedTotal, stats.nackedTotal, stats.releasedTotal, stats.failedTotal, stats.discardedTotal, stats.expiredTotal\n",
				code:   4,
			},
		},
		{
			// The error names the flag as it was typed, the shorthand here.
			name: "unknown column",
			args: []string{"queues", "list", "-s", "main", "-c", "nam"},
			exp: result{
				stderr: "-c: no column matches \"nam\"; available columns: name, uid, createdAt, state, settings.leaseTimeoutSeconds, settings.maxRetries, settings.dedupWindowSeconds, settings.retentionSeconds, settings.effectiveRetentionSeconds, settings.recvRateLimit.intervalMicros, settings.recvRateLimit.burst, stats.sampledAt, stats.startedAt, stats.total, stats.dataBytes, stats.delayed, stats.visible, stats.pending, stats.failed, stats.unavailable, stats.addedTotal, stats.ackedTotal, stats.nackedTotal, stats.releasedTotal, stats.failedTotal, stats.discardedTotal, stats.expiredTotal\n",
				code:   2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var match *regexp.Regexp
			fake := &fakeClient{
				DoQueues: func(_ context.Context, r spooler.QueuesRequest) (spooler.Page[spooler.Queue], error) {
					match = r.Match
					if r.Spool != "main" {
						t.Errorf("spool: %q; want %q", r.Spool, "main")
					}
					return spooler.Page[spooler.Queue]{
						Items: queues,
					}, nil
				},
			}
			act := run(t, fake, "", test.args...)
			expect(t, act, test.exp)
			actMatch := ""
			if match != nil {
				actMatch = match.String()
			}
			if actMatch != test.expMatch {
				t.Errorf("match: %q; want %q", actMatch, test.expMatch)
			}
		})
	}
}
