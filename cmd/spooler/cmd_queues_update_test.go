package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spoolersh/spooler-go"
)

func TestQueuesUpdate(t *testing.T) {
	seconds := func(n int) *int { return &n }
	dir := t.TempDir()
	// A record naming the queue "": an explicit empty name, not an absent one.
	unnamedFile := filepath.Join(dir, "unnamed.json")
	if err := os.WriteFile(unnamedFile, []byte(`{"name":"","settings":{"maxRetries":3}}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		args   []string
		expReq *spooler.UpdateQueueRequest
		exp    result
	}{
		{
			name: "one setting with uid",
			args: []string{"queues", "update", "-s", "main", "-q", "jobs", "--retention", "1d", "--uid", "u1"},
			expReq: &spooler.UpdateQueueRequest{
				Spool:    "main",
				Queue:    "jobs",
				QueueUID: new("u1"),
				Settings: spooler.QueueSettingsUpdate{
					RetentionSeconds: seconds(86400),
				},
			},
		},
		{
			name: "remove the rate limit",
			args: []string{"queues", "update", "-s", "main", "-q", "jobs", "--recv-rate-limit", "unlimited"},
			expReq: &spooler.UpdateQueueRequest{
				Spool: "main",
				Queue: "jobs",
				Settings: spooler.QueueSettingsUpdate{
					RecvRateLimit: &spooler.RateLimit{},
				},
			},
		},
		{
			name: "empty uid goes to the api",
			args: []string{"queues", "update", "-s", "main", "-q", "jobs", "--retention", "1d", "--uid", ""},
			expReq: &spooler.UpdateQueueRequest{
				Spool:    "main",
				Queue:    "jobs",
				QueueUID: new(""),
				Settings: spooler.QueueSettingsUpdate{
					RetentionSeconds: seconds(86400),
				},
			},
		},
		{
			name: "record's empty name goes to the api",
			args: []string{"queues", "update", "-s", "main", "-f", unnamedFile},
			expReq: &spooler.UpdateQueueRequest{
				Spool: "main",
				Queue: "",
				Settings: spooler.QueueSettingsUpdate{
					MaxRetries: seconds(3),
				},
			},
		},
		{
			name: "nothing to update",
			args: []string{"queues", "update", "-s", "main", "-q", "jobs"},
			exp: result{
				stderr: "nothing to update: pass a settings flag or --file\n",
				code:   2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var req *spooler.UpdateQueueRequest
			fake := &fakeClient{
				DoUpdateQueue: func(_ context.Context, r spooler.UpdateQueueRequest) error {
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
