package main

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/spoolersh/spooler-go"
)

func TestQueueSettingsCreate(t *testing.T) {
	seconds := func(n int) *int { return &n }
	interval := func(d time.Duration) *time.Duration { return &d }
	for _, test := range []struct {
		name  string
		flags queueSettingsFlags
		rec   queueSettingsRecord
		exp   spooler.QueueSettingsCreate
		err   bool
	}{
		{
			name: "nothing given",
		},
		{
			name: "record alone",
			rec: queueSettingsRecord{
				LeaseTimeoutSeconds: seconds(300),
				MaxRetries:          seconds(10),
				RecvRateLimit: &rateLimitRecord{
					IntervalMicros: 4000,
					Burst:          10,
				},
			},
			exp: spooler.QueueSettingsCreate{
				LeaseTimeoutSeconds: seconds(300),
				MaxRetries:          seconds(10),
				RecvRateLimit: &spooler.RateLimit{
					IntervalMicros: 4000,
					Burst:          10,
				},
			},
		},
		{
			name: "record's zero rate limit is none",
			rec: queueSettingsRecord{
				RecvRateLimit: &rateLimitRecord{},
			},
		},
		{
			name: "flags over record",
			flags: queueSettingsFlags{
				leaseTimeoutSeconds: seconds(60),
				recvBurst:           seconds(5),
			},
			rec: queueSettingsRecord{
				LeaseTimeoutSeconds: seconds(300),
				RetentionSeconds:    seconds(86400),
				RecvRateLimit: &rateLimitRecord{
					IntervalMicros: 4000,
					Burst:          10,
				},
			},
			exp: spooler.QueueSettingsCreate{
				LeaseTimeoutSeconds: seconds(60),
				RetentionSeconds:    seconds(86400),
				RecvRateLimit: &spooler.RateLimit{
					IntervalMicros: 4000,
					Burst:          5,
				},
			},
		},
		{
			name: "rate limit from flags",
			flags: queueSettingsFlags{
				recvRateInterval: interval(4 * time.Millisecond),
				recvBurst:        seconds(10),
			},
			exp: spooler.QueueSettingsCreate{
				RecvRateLimit: &spooler.RateLimit{
					IntervalMicros: 4000,
					Burst:          10,
				},
			},
		},
		{
			name: "half a rate limit",
			flags: queueSettingsFlags{
				recvBurst: seconds(10),
			},
			err: true,
		},
		{
			name: "unlimited over a record's limit is none",
			flags: queueSettingsFlags{
				recvRateInterval: interval(0),
			},
			rec: queueSettingsRecord{
				RecvRateLimit: &rateLimitRecord{
					IntervalMicros: 4000,
					Burst:          10,
				},
			},
		},
		{
			name: "burst with unlimited",
			flags: queueSettingsFlags{
				recvRateInterval: interval(0),
				recvBurst:        seconds(10),
			},
			err: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			act, err := test.flags.toQueueSettingsCreate(test.rec)
			if test.err && err == nil {
				t.Errorf("want error; got nothing")
			}
			if !test.err && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(test.exp, act); diff != "" {
				t.Errorf("settings:\n%s", diff)
			}
		})
	}
}

func TestQueueSettingsUpdate(t *testing.T) {
	seconds := func(n int) *int { return &n }
	interval := func(d time.Duration) *time.Duration { return &d }
	for _, test := range []struct {
		name  string
		flags queueSettingsFlags
		rec   queueSettingsRecord
		exp   spooler.QueueSettingsUpdate
		err   bool
	}{
		{
			name: "nothing to update",
			err:  true,
		},
		{
			name: "one flag keeps the rest",
			flags: queueSettingsFlags{
				maxRetries: seconds(3),
			},
			exp: spooler.QueueSettingsUpdate{
				MaxRetries: seconds(3),
			},
		},
		{
			name: "record applied as a whole",
			rec: queueSettingsRecord{
				LeaseTimeoutSeconds: seconds(300),
				MaxRetries:          seconds(10),
				DedupWindowSeconds:  seconds(0),
				RetentionSeconds:    seconds(86400),
				RecvRateLimit:       &rateLimitRecord{},
			},
			exp: spooler.QueueSettingsUpdate{
				LeaseTimeoutSeconds: seconds(300),
				MaxRetries:          seconds(10),
				DedupWindowSeconds:  seconds(0),
				RetentionSeconds:    seconds(86400),
				RecvRateLimit:       &spooler.RateLimit{},
			},
		},
		{
			name: "flag over record",
			flags: queueSettingsFlags{
				retentionSeconds: seconds(3600),
			},
			rec: queueSettingsRecord{
				RetentionSeconds: seconds(86400),
			},
			exp: spooler.QueueSettingsUpdate{
				RetentionSeconds: seconds(3600),
			},
		},
		{
			name: "unlimited removes the limit",
			flags: queueSettingsFlags{
				recvRateInterval: interval(0),
			},
			exp: spooler.QueueSettingsUpdate{
				RecvRateLimit: &spooler.RateLimit{},
			},
		},
		{
			name: "burst alone over a record's limit",
			flags: queueSettingsFlags{
				recvBurst: seconds(5),
			},
			rec: queueSettingsRecord{
				RecvRateLimit: &rateLimitRecord{
					IntervalMicros: 4000,
					Burst:          10,
				},
			},
			exp: spooler.QueueSettingsUpdate{
				RecvRateLimit: &spooler.RateLimit{
					IntervalMicros: 4000,
					Burst:          5,
				},
			},
		},
		{
			name: "burst alone without a limit",
			flags: queueSettingsFlags{
				recvBurst: seconds(5),
			},
			err: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			act, err := test.flags.toQueueSettingsUpdate(test.rec)
			if test.err && err == nil {
				t.Errorf("want error; got nothing")
			}
			if !test.err && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(test.exp, act); diff != "" {
				t.Errorf("settings:\n%s", diff)
			}
		})
	}
}

func TestParseWholeSeconds(t *testing.T) {
	for _, test := range []struct {
		name string
		s    string
		exp  int
		err  bool
	}{
		{
			name: "zero",
			s:    "0",
			exp:  0,
		},
		{
			name: "seconds",
			s:    "90s",
			exp:  90,
		},
		{
			name: "two units",
			s:    "1m30s",
			exp:  90,
		},
		{
			name: "days",
			s:    "1d",
			exp:  86400,
		},
		{
			name: "bare number is seconds",
			s:    "90",
			exp:  90,
		},
		{
			name: "fraction of a second",
			s:    "1.5s",
			err:  true,
		},
		{
			name: "bare fraction",
			s:    "1.5",
			err:  true,
		},
		{
			name: "negative",
			s:    "-1s",
			err:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			act, err := parseWholeSeconds(test.s)
			if test.err && err == nil {
				t.Errorf("want error; got nothing")
			}
			if !test.err && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if act != test.exp {
				t.Errorf("seconds: %d; want %d", act, test.exp)
			}
		})
	}
}

func TestParseCount(t *testing.T) {
	for _, test := range []struct {
		name string
		s    string
		exp  int
		err  bool
	}{
		{
			name: "zero",
			s:    "0",
			exp:  0,
		},
		{
			name: "positive",
			s:    "10",
			exp:  10,
		},
		{
			name: "negative",
			s:    "-1",
			err:  true,
		},
		{
			name: "fraction",
			s:    "1.5",
			err:  true,
		},
		{
			name: "word",
			s:    "ten",
			err:  true,
		},
		{
			name: "empty",
			s:    "",
			err:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			act, err := parseCount(test.s)
			if test.err && err == nil {
				t.Errorf("want error; got nothing")
			}
			if !test.err && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if act != test.exp {
				t.Errorf("count: %d; want %d", act, test.exp)
			}
		})
	}
}
