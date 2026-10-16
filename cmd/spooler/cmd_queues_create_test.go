package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spoolersh/spooler-go"
)

func TestQueuesCreate(t *testing.T) {
	seconds := func(n int) *int { return &n }
	dir := t.TempDir()
	// The record is a queue as describe prints it.
	recordFile := filepath.Join(dir, "queue.json")
	described := frames(t, queueOutput{
		Name:  "orders",
		UID:   "u1",
		State: "active",
		Settings: queueSettingsOutput{
			LeaseTimeoutSeconds: 300,
			MaxRetries:          10,
			RecvRateLimit: rateLimitOutput{
				IntervalMicros: 4000,
				Burst:          10,
			},
		},
	})
	if err := os.WriteFile(recordFile, []byte(described), 0600); err != nil {
		t.Fatal(err)
	}
	// A record naming the queue "": an explicit empty name, not an absent one.
	unnamedFile := filepath.Join(dir, "unnamed.json")
	if err := os.WriteFile(unnamedFile, []byte(`{"name":"","settings":{"maxRetries":3}}`), 0600); err != nil {
		t.Fatal(err)
	}
	created := spooler.Queue{
		Name:  "jobs",
		State: spooler.QueueStateActive,
	}
	createdOutput := queueOutput{
		Name:  "jobs",
		State: "active",
	}
	for _, test := range []struct {
		name   string
		args   []string
		expReq *spooler.CreateQueueRequest
		exp    result
	}{
		{
			name: "flags",
			args: []string{"queues", "create", "-s", "main", "-q", "jobs", "--lease-timeout", "5m", "--max-retries", "3"},
			expReq: &spooler.CreateQueueRequest{
				Spool: "main",
				Queue: "jobs",
				Settings: spooler.QueueSettingsCreate{
					LeaseTimeoutSeconds: seconds(300),
					MaxRetries:          seconds(3),
				},
			},
			exp: result{
				stdout: record(t, createdOutput),
			},
		},
		{
			name: "record with a flag over it",
			args: []string{"queues", "create", "-s", "main", "-f", recordFile, "--recv-burst", "5"},
			expReq: &spooler.CreateQueueRequest{
				Spool: "main",
				Queue: "orders",
				Settings: spooler.QueueSettingsCreate{
					LeaseTimeoutSeconds: seconds(300),
					MaxRetries:          seconds(10),
					DedupWindowSeconds:  seconds(0),
					RecvRateLimit: &spooler.RateLimit{
						IntervalMicros: 4000,
						Burst:          5,
					},
				},
			},
			exp: result{
				stdout: record(t, createdOutput),
			},
		},
		{
			name: "queue flag over the record's name",
			args: []string{"queues", "create", "-s", "main", "-f", recordFile, "-q", "jobs"},
			expReq: &spooler.CreateQueueRequest{
				Spool: "main",
				Queue: "jobs",
				Settings: spooler.QueueSettingsCreate{
					LeaseTimeoutSeconds: seconds(300),
					MaxRetries:          seconds(10),
					DedupWindowSeconds:  seconds(0),
					RecvRateLimit: &spooler.RateLimit{
						IntervalMicros: 4000,
						Burst:          10,
					},
				},
			},
			exp: result{
				stdout: record(t, createdOutput),
			},
		},
		{
			name: "record's empty name goes to the api",
			args: []string{"queues", "create", "-s", "main", "-f", unnamedFile},
			expReq: &spooler.CreateQueueRequest{
				Spool: "main",
				Queue: "",
				Settings: spooler.QueueSettingsCreate{
					MaxRetries: seconds(3),
				},
			},
			exp: result{
				stdout: record(t, createdOutput),
			},
		},
		{
			name: "no queue",
			args: []string{"queues", "create", "-s", "main"},
			exp: result{
				stderr: "no queue: pass --queue or --file\n",
				code:   2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var req *spooler.CreateQueueRequest
			fake := &fakeClient{
				DoCreateQueue: func(_ context.Context, r spooler.CreateQueueRequest) (spooler.Queue, error) {
					req = &r
					return created, nil
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
