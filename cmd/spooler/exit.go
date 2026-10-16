package main

import (
	"context"
	"errors"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-go"
)

func mapExit(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, cli.ErrHelp) {
		// The runner answers these itself with exit 0.
		return err
	}
	if _, ok := errors.AsType[*cli.UsageError](err); ok {
		// The runner answers these itself with exit 2.
		return err
	}
	if _, ok := errors.AsType[*cli.ExitError](err); ok {
		return err
	}
	if _, ok := errors.AsType[*cli.SignalError](context.Cause(ctx)); ok {
		if !errors.Is(err, spooler.ErrResultUnknown) {
			return err
		}
	}
	// Wrapped with %w, so the cause stays reachable with errors.Is.
	return cli.Exitf(exitCode(err), "%w", err)
}

// exitCode is the exit code for err, by the following table:
//
//	100 ErrResultUnknown: the change may have happened; checked first,
//	    since a later attempt's answer never clears an earlier unknown,
//	    and numbered apart from the rest since it is the one outcome a
//	    caller must think about rather than retry or give up
//
//	8   retry later: ErrUnavailable, or a kind the docs list as transient
//	    (rate_limited, queue_creating, queue_busy, dedup_in_flight)
//
//	4   failed, do not retry: any other API error, a cancelled or expired
//	    context (one that expired with the request on the wire is marked
//	    ErrResultUnknown by the SDK and is 100 above), and every local
//	    failure (a file, a broken pipe, a template that fails to execute,
//	    a transport that reached no server)
//
//	2   ErrInvalidRequest: a request the SDK refused before sending,
//	    the empty API key among them
func exitCode(err error) int {
	switch {
	case
		errors.Is(err, spooler.ErrResultUnknown):
		return 100

	case
		errors.Is(err, spooler.ErrUnavailable),
		errors.Is(err, spooler.ErrRateLimited),
		errors.Is(err, spooler.ErrQueueCreating),
		errors.Is(err, spooler.ErrQueueBusy),
		errors.Is(err, spooler.ErrDedupInFlight):
		return 8

	case
		errors.Is(err, spooler.ErrInvalidRequest):
		return 2

	default:
		return 4
	}
}
