package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/spoolersh/spooler-go"
)

// TestMessagesPageFailure pins that a listing cut by a failed page prints
// the messages fetched before it whole, in the buffering formats too, then
// the error.
func TestMessagesPageFailure(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		exp  result
	}{
		{
			name: "csv",
			args: []string{"messages", "-s", "main", "-q", "jobs", "-o", "csv", "-c", "id,state"},
			exp: result{
				stdout: "id,state\nm1,visible\nm2,visible\n",
				stderr: "spooler: unavailable\n",
				code:   8,
			},
		},
		{
			name: "table",
			args: []string{"messages", "-s", "main", "-q", "jobs", "-o", "table", "-c", "id,state"},
			exp: result{
				stdout: "ID  STATE\nm1  visible\nm2  visible\n",
				stderr: "spooler: unavailable\n",
				code:   8,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeClient{
				DoMessages: func(_ context.Context, r spooler.MessagesRequest) (spooler.Page[spooler.Message], error) {
					if r.After != "" {
						return spooler.Page[spooler.Message]{}, fmt.Errorf(
							"spooler: %w",
							spooler.ErrUnavailable,
						)
					}
					return spooler.Page[spooler.Message]{
						Items: []spooler.Message{
							{
								ID:    "m1",
								State: spooler.MessageStateVisible,
							},
							{
								ID:    "m2",
								State: spooler.MessageStateVisible,
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

func TestMessages(t *testing.T) {
	created := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	// Three messages over two pages of two.
	page := func(after string) spooler.Page[spooler.Message] {
		if after == "" {
			return spooler.Page[spooler.Message]{
				Items: []spooler.Message{
					{
						ID:        "m1",
						State:     spooler.MessageStateVisible,
						CreatedAt: created,
					},
					{
						ID:        "m2",
						State:     spooler.MessageStateVisible,
						CreatedAt: created,
					},
				},
				Next: "c1",
			}
		}
		return spooler.Page[spooler.Message]{
			Items: []spooler.Message{
				{
					ID:        "m3",
					State:     spooler.MessageStatePending,
					CreatedAt: created,
				},
			},
		}
	}
	m1 := messageOutput{
		ID:        "m1",
		State:     "visible",
		CreatedAt: created,
	}
	m2 := messageOutput{
		ID:        "m2",
		State:     "visible",
		CreatedAt: created,
	}
	m3 := messageOutput{
		ID:        "m3",
		State:     "pending",
		CreatedAt: created,
	}
	for _, test := range []struct {
		name    string
		args    []string
		expReqs []spooler.MessagesRequest
		exp     result
	}{
		{
			name: "every page as ndjson",
			args: []string{"messages", "-s", "main", "-q", "jobs", "-o", "json", "--compact"},
			expReqs: []spooler.MessagesRequest{
				{
					Spool: "main",
					Queue: "jobs",
				},
				{
					Spool: "main",
					Queue: "jobs",
					After: "c1",
				},
			},
			exp: result{
				stdout: frames(t, m1, m2, m3),
			},
		},
		{
			name: "limit stops at the first page",
			args: []string{"messages", "-s", "main", "-q", "jobs", "-o", "json", "--compact", "-n", "2"},
			expReqs: []spooler.MessagesRequest{
				{
					Spool: "main",
					Queue: "jobs",
				},
			},
			exp: result{
				stdout: frames(t, m1, m2),
			},
		},
		{
			name: "state and handle are passed on",
			args: []string{"messages", "-s", "main", "-q", "jobs", "-o", "json", "--compact", "--state", "pending", "--return-handle", "-n", "1"},
			expReqs: []spooler.MessagesRequest{
				{
					Spool:        "main",
					Queue:        "jobs",
					State:        spooler.MessageStatePending,
					ReturnHandle: true,
				},
			},
			exp: result{
				stdout: frames(t, m1),
			},
		},
		{
			name: "unknown state",
			args: []string{"messages", "-s", "main", "-q", "jobs", "--state", "lost"},
			exp: result{
				stderr: "--state: unknown message state \"lost\"\n",
				code:   2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var reqs []spooler.MessagesRequest
			fake := &fakeClient{
				DoMessages: func(_ context.Context, r spooler.MessagesRequest) (spooler.Page[spooler.Message], error) {
					reqs = append(reqs, r)
					return page(r.After), nil
				},
			}
			act := run(t, fake, "", test.args...)
			expect(t, act, test.exp)
			if diff := cmp.Diff(test.expReqs, reqs); diff != "" {
				t.Errorf("requests:\n%s", diff)
			}
		})
	}
}
