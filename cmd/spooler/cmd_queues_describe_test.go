package main

import (
	"context"
	"testing"

	"github.com/spoolersh/spooler-go"
)

func TestQueuesDescribe(t *testing.T) {
	queue := spooler.Queue{
		Name:  "jobs",
		UID:   "u1",
		State: spooler.QueueStateActive,
		Settings: spooler.QueueSettings{
			LeaseTimeoutSeconds: 300,
			RetentionSeconds:    86400,
			RecvRateLimit: spooler.RateLimit{
				IntervalMicros: 4000,
				Burst:          10,
			},
		},
		Stats: spooler.MessageStats{
			DataBytes: 1536 << 10,
		},
	}
	described := queueOutput{
		Name:  "jobs",
		UID:   "u1",
		State: "active",
		Settings: queueSettingsOutput{
			LeaseTimeoutSeconds: 300,
			RetentionSeconds:    86400,
			RecvRateLimit: rateLimitOutput{
				IntervalMicros: 4000,
				Burst:          10,
			},
		},
		Stats: messageStatsOutput{
			DataBytes: 1536 << 10,
		},
	}
	for _, test := range []struct {
		name string
		args []string
		err  error
		exp  result
	}{
		{
			name: "record",
			args: []string{"queues", "describe", "-s", "main", "-q", "jobs"},
			exp: result{
				stdout: record(t, described),
			},
		},
		{
			name: "human needs text or table",
			args: []string{"queues", "describe", "-s", "main", "-q", "jobs", "--human"},
			exp: result{
				stderr: "--human takes effect only with `-o table` or `-o text`\n",
				code:   2,
			},
		},
		{
			name: "not found",
			args: []string{"queues", "describe", "-s", "main", "-q", "jobs"},
			err:  apiError(spooler.ErrorKindQueueNotFound, "queue jobs not found"),
			exp: result{
				stderr: "queue jobs not found\n",
				code:   4,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeClient{
				DoQueue: func(_ context.Context, r spooler.QueueRequest) (spooler.Queue, error) {
					if r.Spool != "main" || r.Queue != "jobs" {
						t.Errorf("request: %s/%s; want main/jobs", r.Spool, r.Queue)
					}
					return queue, test.err
				},
			}
			act := run(t, fake, "", test.args...)
			expect(t, act, test.exp)
		})
	}
}
