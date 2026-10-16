package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/spoolersh/spooler-go"
)

func TestFormatSpoolJSON(t *testing.T) {
	for _, test := range []struct {
		name  string
		spool spooler.Spool
		exp   string
	}{
		{
			name: "counts and totals",
			spool: spooler.Spool{
				Name:       "main",
				QueueCount: 2,
				Stats: spooler.MessageStats{
					SampledAt:  time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC),
					StartedAt:  time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
					Total:      5,
					Visible:    3,
					Pending:    2,
					AddedTotal: 7,
					AckedTotal: 2,
				},
			},
			exp: `{"name":"main","queueCount":2,"stats":{"sampledAt":"2026-10-15T09:00:00Z","startedAt":"2026-10-01T00:00:00Z","total":5,"dataBytes":0,"delayed":0,"visible":3,"pending":2,"failed":0,"unavailable":0,"addedTotal":7,"ackedTotal":2,"nackedTotal":0,"releasedTotal":0,"failedTotal":0,"discardedTotal":0,"expiredTotal":0}}`,
		},
		{
			name: "empty spool has no sample times",
			spool: spooler.Spool{
				Name: "main",
			},
			exp: `{"name":"main","queueCount":0,"stats":{"total":0,"dataBytes":0,"delayed":0,"visible":0,"pending":0,"failed":0,"unavailable":0,"addedTotal":0,"ackedTotal":0,"nackedTotal":0,"releasedTotal":0,"failedTotal":0,"discardedTotal":0,"expiredTotal":0}}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			bts, err := json.Marshal(formatSpool(test.spool))
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if act := string(bts); act != test.exp {
				t.Errorf(
					"json: %s; want %s",
					act, test.exp,
				)
			}
		})
	}
}

func TestFormatMessageJSON(t *testing.T) {
	for _, test := range []struct {
		name string
		msg  spooler.Message
		exp  string
	}{
		{
			name: "every field",
			msg: spooler.Message{
				ID:             "m1",
				State:          spooler.MessageStatePending,
				Retries:        2,
				CreatedAt:      time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC),
				VisibleAt:      time.Date(2026, 10, 13, 9, 5, 0, 0, time.UTC),
				LeaseExpiresAt: time.Date(2026, 10, 13, 9, 35, 0, 0, time.UTC),
				ExpiresAt:      time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC),
				Handle:         "H1",
			},
			exp: `{"id":"m1","state":"pending","retries":2,"createdAt":"2026-10-13T09:00:00Z","visibleAt":"2026-10-13T09:05:00Z","leaseExpiresAt":"2026-10-13T09:35:00Z","expiresAt":"2026-10-14T09:00:00Z","handle":"H1"}`,
		},
		{
			name: "optional fields are absent",
			msg: spooler.Message{
				ID:        "m2",
				State:     spooler.MessageStateVisible,
				CreatedAt: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC),
			},
			exp: `{"id":"m2","state":"visible","retries":0,"createdAt":"2026-10-13T09:00:00Z"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			bts, err := json.Marshal(formatMessage(test.msg))
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if act := string(bts); act != test.exp {
				t.Errorf(
					"json: %s; want %s",
					act, test.exp,
				)
			}
		})
	}
}

func TestFormatRecvResultJSON(t *testing.T) {
	for _, test := range []struct {
		name string
		res  spooler.RecvResult
		exp  string
	}{
		{
			name: "every field",
			res: spooler.RecvResult{
				ID:             "m1",
				Retries:        2,
				CreatedAt:      time.Date(2026, 10, 13, 9, 30, 0, 0, time.UTC),
				Lease:          "L1",
				LeaseExpiresAt: time.Date(2026, 10, 13, 9, 35, 0, 0, time.UTC),
				DataSize:       42,
			},
			exp: `{"id":"m1","retries":2,"createdAt":"2026-10-13T09:30:00Z","lease":"L1","leaseExpiresAt":"2026-10-13T09:35:00Z","dataSize":42}`,
		},
		{
			name: "empty payload is zero bytes",
			res: spooler.RecvResult{
				ID:        "m2",
				CreatedAt: time.Date(2026, 10, 13, 9, 30, 0, 0, time.UTC),
				Lease:     "L2",
				DataSize:  0,
			},
			exp: `{"id":"m2","retries":0,"createdAt":"2026-10-13T09:30:00Z","lease":"L2","dataSize":0}`,
		},
		{
			name: "undeclared size and no expiry are absent",
			res: spooler.RecvResult{
				ID:        "m3",
				CreatedAt: time.Date(2026, 10, 13, 9, 30, 0, 0, time.UTC),
				Lease:     "L3",
				DataSize:  -1,
			},
			exp: `{"id":"m3","retries":0,"createdAt":"2026-10-13T09:30:00Z","lease":"L3"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			bts, err := json.Marshal(formatRecvResult(test.res))
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if act := string(bts); act != test.exp {
				t.Errorf(
					"json: %s; want %s",
					act, test.exp,
				)
			}
		})
	}
}

func TestFormatRenewResultJSON(t *testing.T) {
	for _, test := range []struct {
		name string
		res  spooler.RenewResult
		exp  string
	}{
		{
			name: "new expiry",
			res: spooler.RenewResult{
				LeaseExpiresAt: time.Date(2026, 10, 13, 9, 35, 0, 0, time.UTC),
			},
			exp: `{"leaseExpiresAt":"2026-10-13T09:35:00Z"}`,
		},
		{
			name: "no expiry is absent",
			res:  spooler.RenewResult{},
			exp:  `{}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			bts, err := json.Marshal(formatRenewResult(test.res))
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if act := string(bts); act != test.exp {
				t.Errorf(
					"json: %s; want %s",
					act, test.exp,
				)
			}
		})
	}
}

func TestFormatInspectResultJSON(t *testing.T) {
	for _, test := range []struct {
		name string
		res  spooler.InspectResult
		exp  string
	}{
		{
			name: "every field",
			res: spooler.InspectResult{
				ID:        "m1",
				State:     spooler.MessageStateFailed,
				Retries:   3,
				CreatedAt: time.Date(2026, 10, 13, 9, 30, 0, 0, time.UTC),
				ExpiresAt: time.Date(2026, 10, 14, 9, 30, 0, 0, time.UTC),
				DataSize:  42,
			},
			exp: `{"id":"m1","state":"failed","retries":3,"createdAt":"2026-10-13T09:30:00Z","expiresAt":"2026-10-14T09:30:00Z","dataSize":42}`,
		},
		{
			name: "pending carries the lease expiry",
			res: spooler.InspectResult{
				ID:             "m2",
				State:          spooler.MessageStatePending,
				Retries:        1,
				CreatedAt:      time.Date(2026, 10, 13, 9, 30, 0, 0, time.UTC),
				LeaseExpiresAt: time.Date(2026, 10, 13, 9, 35, 0, 0, time.UTC),
				DataSize:       42,
			},
			exp: `{"id":"m2","state":"pending","retries":1,"createdAt":"2026-10-13T09:30:00Z","leaseExpiresAt":"2026-10-13T09:35:00Z","dataSize":42}`,
		},
		{
			name: "delayed carries when it becomes visible",
			res: spooler.InspectResult{
				ID:        "m3",
				State:     spooler.MessageStateDelayed,
				CreatedAt: time.Date(2026, 10, 13, 9, 30, 0, 0, time.UTC),
				VisibleAt: time.Date(2026, 10, 13, 9, 40, 0, 0, time.UTC),
				DataSize:  42,
			},
			exp: `{"id":"m3","state":"delayed","retries":0,"createdAt":"2026-10-13T09:30:00Z","visibleAt":"2026-10-13T09:40:00Z","dataSize":42}`,
		},
		{
			name: "undeclared size is absent",
			res: spooler.InspectResult{
				ID:        "m2",
				State:     spooler.MessageStateDelayed,
				CreatedAt: time.Date(2026, 10, 13, 9, 30, 0, 0, time.UTC),
				DataSize:  -1,
			},
			exp: `{"id":"m2","state":"delayed","retries":0,"createdAt":"2026-10-13T09:30:00Z"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			bts, err := json.Marshal(formatInspectResult(test.res))
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if act := string(bts); act != test.exp {
				t.Errorf(
					"json: %s; want %s",
					act, test.exp,
				)
			}
		})
	}
}
