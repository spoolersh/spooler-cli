package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-go"
)

func TestMapExit(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		// signal, when set, has cancelled the context.
		signal os.Signal
		// asIs expects the error back unchanged, without a code.
		asIs bool
		code int
		msg  string
	}{
		{
			name: "usage error keeps its code",
			err:  cli.Exitf(2, "no queue: pass --queue"),
			code: 2,
			msg:  "no queue: pass --queue",
		},
		{
			name: "silent one keeps its code",
			err: &cli.ExitError{
				Code: 1,
			},
			code: 1,
		},
		{
			name: "help is the runner's",
			err:  cli.ErrHelp,
			asIs: true,
		},
		{
			name: "usage is the runner's",
			err:  new(cli.UsageError),
			asIs: true,
		},
		{
			name: "cancellation",
			err:  fmt.Errorf("send: %w", context.Canceled),
			code: 4,
			msg:  "send: context canceled",
		},
		{
			name: "deadline exceeded",
			err:  fmt.Errorf("recv: %w", context.DeadlineExceeded),
			code: 4,
			msg:  "recv: context deadline exceeded",
		},
		{
			name: "deadline exceeded with the request on the wire",
			err: errors.Join(
				context.DeadlineExceeded,
				spooler.ErrResultUnknown,
			),
			code: 100,
		},
		{
			name: "result unknown",
			err:  fmt.Errorf("spooler: %w", spooler.ErrResultUnknown),
			code: 100,
			msg:  "spooler: result unknown",
		},
		{
			name: "result unknown wins over a transient kind",
			err: errors.Join(
				spooler.ErrRateLimited,
				spooler.ErrResultUnknown,
			),
			code: 100,
		},
		{
			name: "unavailable",
			err:  fmt.Errorf("spooler: %w", spooler.ErrUnavailable),
			code: 8,
			msg:  "spooler: unavailable",
		},
		{
			name: "rate limited",
			err: &spooler.Error{
				Kind:    spooler.ErrorKindRateLimited,
				Message: "slow down",
			},
			code: 8,
			msg:  "slow down",
		},
		{
			name: "queue creating",
			err: &spooler.Error{
				Kind: spooler.ErrorKindQueueCreating,
			},
			code: 8,
		},
		{
			name: "queue busy",
			err: &spooler.Error{
				Kind: spooler.ErrorKindQueueBusy,
			},
			code: 8,
		},
		{
			name: "dedup in flight",
			err: &spooler.Error{
				Kind: spooler.ErrorKindDedupInFlight,
			},
			code: 8,
		},
		{
			name: "other api error",
			err: &spooler.Error{
				Kind:    spooler.ErrorKindQueueNotFound,
				Message: "queue jobs not found",
			},
			code: 4,
			msg:  "queue jobs not found",
		},
		{
			name: "invalid request",
			err:  fmt.Errorf("spooler: %w: spool name is required", spooler.ErrInvalidRequest),
			code: 2,
			msg:  "spooler: invalid request: spool name is required",
		},
		{
			name: "local failure",
			err:  fmt.Errorf("open lease.json: %w", os.ErrNotExist),
			code: 4,
			msg:  "open lease.json: file does not exist",
		},
		{
			name:   "signal leaves an error uncoded",
			err:    fmt.Errorf("recv: %w", context.Canceled),
			signal: syscall.SIGINT,
			asIs:   true,
		},
		{
			name:   "signal leaves a coded error coded",
			err:    cli.Exitf(4, "write payload: %w; release: slow down", context.Canceled),
			signal: syscall.SIGINT,
			code:   4,
			msg:    "write payload: context canceled; release: slow down",
		},
		{
			name:   "signal keeps an unknown result",
			err:    fmt.Errorf("spooler: %w", spooler.ErrResultUnknown),
			signal: syscall.SIGTERM,
			code:   100,
			msg:    "spooler: result unknown",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(t.Context())
			defer cancel(nil)
			if test.signal != nil {
				cancel(&cli.SignalError{
					Signal: test.signal,
				})
			}
			act := mapExit(ctx, test.err)
			if test.asIs {
				if act != test.err {
					t.Errorf("error: %v; want %v as it is", act, test.err)
				}
				return
			}
			exit, coded := errors.AsType[*cli.ExitError](act)
			if !coded {
				t.Fatalf("error: %v; want exit %d", act, test.code)
			}
			if exit.Code != test.code {
				t.Errorf("exit: %d; want %d", exit.Code, test.code)
			}
			if !errors.Is(act, test.err) {
				t.Errorf("error: %v; want it to wrap %v", act, test.err)
			}
			if test.msg != "" {
				if msg := act.Error(); msg != test.msg {
					t.Errorf("message: %q; want %q", msg, test.msg)
				}
			}
		})
	}
}

func TestMapExitNil(t *testing.T) {
	if err := mapExit(t.Context(), nil); err != nil {
		t.Errorf("error: %v; want nothing", err)
	}
}
