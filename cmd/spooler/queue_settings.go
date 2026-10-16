package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-cli/internal/flagvars"
	"github.com/spoolersh/spooler-cli/internal/human"
	"github.com/spoolersh/spooler-go"
)

// queueRecord is a queue as `queues describe -o json` prints it, read back
// by --file: its name, absent when the record has none, and its settings.
// The uid, state and stats are ignored, so a describe's output creates a
// queue like the one described.
type queueRecord struct {
	Name     *string             `json:"name"`
	Settings queueSettingsRecord `json:"settings"`
}

// queueSettingsRecord is a record's settings, each absent when the record
// does not have it.
type queueSettingsRecord struct {
	LeaseTimeoutSeconds *int             `json:"leaseTimeoutSeconds"`
	MaxRetries          *int             `json:"maxRetries"`
	DedupWindowSeconds  *int             `json:"dedupWindowSeconds"`
	RetentionSeconds    *int             `json:"retentionSeconds"`
	RecvRateLimit       *rateLimitRecord `json:"recvRateLimit"`
}

// rateLimitRecord is a record's rate limit; all zero means none.
type rateLimitRecord struct {
	IntervalMicros int64 `json:"intervalMicros"`
	Burst          int   `json:"burst"`
}

// readQueueRecord reads the first JSON object in the file as a queue record;
// "-" is stdin.
func readQueueRecord(name string) (queueRecord, error) {
	f := os.Stdin
	if name != "-" {
		var err error
		f, err = os.Open(name)
		if err != nil {
			return queueRecord{}, err
		}
		defer f.Close()
	}
	var rec queueRecord
	if err := json.NewDecoder(f).Decode(&rec); err != nil {
		return queueRecord{}, fmt.Errorf(
			"decode %s: %w",
			name, err,
		)
	}
	// A queue whose retention was never set describes it as 0, which the
	// API does not take (its minimum is 60): the record has none, and the
	// server applies its default, as the describe's effective value shows.
	if r := rec.Settings.RetentionSeconds; r != nil && *r == 0 {
		rec.Settings.RetentionSeconds = nil
	}
	return rec, nil
}

// queueSettingsFlags are the settings flags of queues create and update,
// each nil until given, so a value from --file shows through where no flag
// is.
type queueSettingsFlags struct {
	leaseTimeoutSeconds *int
	maxRetries          *int
	dedupWindowSeconds  *int
	retentionSeconds    *int
	recvRateInterval    *time.Duration
	recvBurst           *int
}

func (s *queueSettingsFlags) define(fs *flag.FlagSet) {
	fs.Var(flagvars.OptVar(&s.leaseTimeoutSeconds, parseWholeSeconds),
		"lease-timeout",
		"`duration` before an unacked delivery is auto-nacked (5m, 300s);"+
			" 0 disables the timeout; omitted means the default on create and"+
			" the current value on update",
	)
	fs.Var(flagvars.OptVar(&s.maxRetries, parseCount),
		"max-retries",
		"`count` of redeliveries before a message is failed; 0 means none;"+
			" omitted means the default on create and the current value on"+
			" update",
	)
	fs.Var(flagvars.OptVar(&s.dedupWindowSeconds, parseWholeSeconds),
		"dedup-window",
		"`duration` a dedup key is remembered (60s); 0 disables dedup;"+
			" omitted means the default on create and the current value on"+
			" update",
	)
	fs.Var(flagvars.OptVar(&s.retentionSeconds, parseWholeSeconds),
		"retention",
		"`duration` a message is kept before it is discarded (1d, 12h);"+
			" omitted means the default on create and the current value on"+
			" update",
	)
	fs.Var(flagvars.OptVar(&s.recvRateInterval, parseRate),
		"recv-rate-limit",
		"receive `rate` as a count per period (100/s, 60/m), with"+
			" --recv-burst, or unlimited to remove the limit; omitted means no"+
			" limit on create and the current value on update",
	)
	fs.Var(flagvars.OptVar(&s.recvBurst, parseCount),
		"recv-burst",
		"`count` of messages an idle queue may release at once, with"+
			" --recv-rate-limit",
	)
}

// toQueueSettingsCreate merges the flags over a record's settings, a flag
// given winning, into a create request's settings. A limit removed, by
// unlimited or by a record's all-zero limit, is no limit.
func (s *queueSettingsFlags) toQueueSettingsCreate(rec queueSettingsRecord) (spooler.QueueSettingsCreate, error) {
	limit, err := s.rateLimit(rec.RecvRateLimit)
	if err != nil {
		return spooler.QueueSettingsCreate{}, err
	}
	if limit != nil && *limit == (spooler.RateLimit{}) {
		limit = nil
	}
	return spooler.QueueSettingsCreate{
		LeaseTimeoutSeconds: cmp.Or(s.leaseTimeoutSeconds, rec.LeaseTimeoutSeconds),
		MaxRetries:          cmp.Or(s.maxRetries, rec.MaxRetries),
		DedupWindowSeconds:  cmp.Or(s.dedupWindowSeconds, rec.DedupWindowSeconds),
		RetentionSeconds:    cmp.Or(s.retentionSeconds, rec.RetentionSeconds),
		RecvRateLimit:       limit,
	}, nil
}

// toQueueSettingsUpdate merges the flags over a record's settings, a flag
// given winning, into an update request's settings: a setting neither gives
// keeps its current value, and a limit removed, by unlimited or by a
// record's all-zero limit, is the zero limit the SDK takes as removal.
// Nothing to set is a usage error.
func (s *queueSettingsFlags) toQueueSettingsUpdate(rec queueSettingsRecord) (spooler.QueueSettingsUpdate, error) {
	limit, err := s.rateLimit(rec.RecvRateLimit)
	if err != nil {
		return spooler.QueueSettingsUpdate{}, err
	}
	upd := spooler.QueueSettingsUpdate{
		LeaseTimeoutSeconds: cmp.Or(s.leaseTimeoutSeconds, rec.LeaseTimeoutSeconds),
		MaxRetries:          cmp.Or(s.maxRetries, rec.MaxRetries),
		DedupWindowSeconds:  cmp.Or(s.dedupWindowSeconds, rec.DedupWindowSeconds),
		RetentionSeconds:    cmp.Or(s.retentionSeconds, rec.RetentionSeconds),
		RecvRateLimit:       limit,
	}
	if upd == (spooler.QueueSettingsUpdate{}) {
		return upd, cli.Exitf(2,
			"nothing to update: pass a settings flag or --file",
		)
	}
	return upd, nil
}

// rateLimit merges the two rate limit flags over a record's limit: the
// flags given win, a record's all-zero limit and the flag value unlimited
// are the zero limit, and one flag without the other and without a record
// to complete it is a usage error.
func (s *queueSettingsFlags) rateLimit(rec *rateLimitRecord) (*spooler.RateLimit, error) {
	var limit *spooler.RateLimit
	if rec != nil {
		limit = &spooler.RateLimit{
			IntervalMicros: rec.IntervalMicros,
			Burst:          rec.Burst,
		}
	}
	if s.recvRateInterval != nil && *s.recvRateInterval == 0 {
		if s.recvBurst != nil {
			return nil, cli.Exitf(2,
				"--recv-burst has no effect with --recv-rate-limit unlimited",
			)
		}
		return new(spooler.RateLimit), nil
	}
	if s.recvRateInterval == nil && s.recvBurst == nil {
		return limit, nil
	}
	if limit == nil || *limit == (spooler.RateLimit{}) {
		if s.recvRateInterval == nil || s.recvBurst == nil {
			return nil, cli.Exitf(2,
				"pass both --recv-rate-limit and --recv-burst, or neither",
			)
		}
		limit = new(spooler.RateLimit)
	}
	if s.recvRateInterval != nil {
		limit.IntervalMicros = int64(*s.recvRateInterval / time.Microsecond)
	}
	if s.recvBurst != nil {
		limit.Burst = *s.recvBurst
	}
	return limit, nil
}

// parseRate reads a rate (100/s) whose interval is whole microseconds, as
// the API takes it, or unlimited as the zero interval.
func parseRate(s string) (time.Duration, error) {
	if s == "unlimited" {
		return 0, nil
	}
	d, err := human.ParseRate(s)
	if err != nil {
		return 0, err
	}
	if d%time.Microsecond != 0 {
		return 0, fmt.Errorf(
			"rate %q: interval is not whole microseconds",
			s,
		)
	}
	return d, nil
}

// parseWholeSeconds reads a duration (5m, 1d, 300s) that is whole seconds,
// as the API takes it, or a bare number of seconds (300), as sleep and
// gcloud read one, and returns the seconds.
func parseWholeSeconds(s string) (int, error) {
	if n, err := strconv.ParseUint(s, 10, strconv.IntSize-1); err == nil {
		return int(n), nil
	}
	d, err := human.ParseDuration(s)
	if _, bare := strconv.ParseFloat(s, 64); bare != nil && err != nil {
		return 0, err
	}
	if err != nil || d < 0 || d%time.Second != 0 {
		return 0, fmt.Errorf(
			"duration %q: want whole seconds, as 90 or 1m30s",
			s,
		)
	}
	return int(d / time.Second), nil
}

// parseCount reads a non-negative integer.
func parseCount(s string) (int, error) {
	n, err := strconv.ParseUint(s, 10, strconv.IntSize-1)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}
