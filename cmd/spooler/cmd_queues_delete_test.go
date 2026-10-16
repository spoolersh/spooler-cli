package main

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spoolersh/spooler-go"
)

func TestQueuesDelete(t *testing.T) {
	for _, test := range []struct {
		name   string
		args   []string
		err    error
		expReq *spooler.DeleteQueueRequest
		exp    result
	}{
		{
			name: "force",
			args: []string{"queues", "delete", "-s", "main", "-q", "jobs", "--mode", "force", "--yes"},
			expReq: &spooler.DeleteQueueRequest{
				Spool: "main",
				Queue: "jobs",
				Mode:  spooler.QueueDeleteModeForce,
			},
		},
		{
			name: "drain with uid",
			args: []string{"queues", "delete", "-s", "main", "-q", "jobs", "--mode", "drain", "--uid", "u1", "--yes"},
			expReq: &spooler.DeleteQueueRequest{
				Spool:    "main",
				Queue:    "jobs",
				QueueUID: new("u1"),
				Mode:     spooler.QueueDeleteModeDrain,
			},
		},
		{
			name: "no mode",
			args: []string{"queues", "delete", "-s", "main", "-q", "jobs", "--yes"},
			exp: result{
				stderr: "no mode: pass --mode force to delete the queue and its messages at once, or --mode drain to delete it once consumers have emptied it\n",
				code:   2,
			},
		},
		{
			name: "empty uid goes to the api",
			args: []string{"queues", "delete", "-s", "main", "-q", "jobs", "--mode", "force", "--uid", "", "--yes"},
			expReq: &spooler.DeleteQueueRequest{
				Spool:    "main",
				Queue:    "jobs",
				QueueUID: new(""),
				Mode:     spooler.QueueDeleteModeForce,
			},
		},
		{
			name: "without consent",
			args: []string{"queues", "delete", "-s", "main", "-q", "jobs", "--mode", "force"},
			exp: result{
				stderr: "queues delete removes the queue \"jobs\" for good: pass --yes to confirm\n",
				code:   2,
			},
		},
		{
			name: "replaced since",
			args: []string{"queues", "delete", "-s", "main", "-q", "jobs", "--mode", "force", "--uid", "u1", "--yes"},
			err:  apiError(spooler.ErrorKindQueueReplaced, "queue jobs was replaced"),
			expReq: &spooler.DeleteQueueRequest{
				Spool:    "main",
				Queue:    "jobs",
				QueueUID: new("u1"),
				Mode:     spooler.QueueDeleteModeForce,
			},
			exp: result{
				stderr: "queue jobs was replaced\n",
				code:   4,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var req *spooler.DeleteQueueRequest
			fake := &fakeClient{
				DoDeleteQueue: func(_ context.Context, r spooler.DeleteQueueRequest) error {
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
