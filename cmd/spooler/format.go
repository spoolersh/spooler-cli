package main

import (
	"time"

	"github.com/spoolersh/spooler-go"
)

// The *Output types are what the commands print: the SDK's results with the
// docs' field names, rendered by output[T] in every format.

// queueOutput is a queue as the CLI prints it, with the docs' field names
// (https://docs.spooler.sh/operations/spoolQueue).
type queueOutput struct {
	Name      string              `json:"name"`
	UID       string              `json:"uid"`
	CreatedAt time.Time           `json:"createdAt" human:"age"`
	State     string              `json:"state"`
	Settings  queueSettingsOutput `json:"settings"`
	Stats     messageStatsOutput  `json:"stats"`
}

// queueSettingsOutput is a queue's settings as applied.
type queueSettingsOutput struct {
	LeaseTimeoutSeconds       int             `json:"leaseTimeoutSeconds" human:"seconds"`
	MaxRetries                int             `json:"maxRetries"`
	DedupWindowSeconds        int             `json:"dedupWindowSeconds" human:"seconds"`
	RetentionSeconds          int             `json:"retentionSeconds" human:"seconds"`
	EffectiveRetentionSeconds int             `json:"effectiveRetentionSeconds" human:"seconds"`
	RecvRateLimit             rateLimitOutput `json:"recvRateLimit"`
}

// rateLimitOutput is a token bucket rate limit; all zero means no limit.
type rateLimitOutput struct {
	IntervalMicros int64 `json:"intervalMicros" human:"rate"`
	Burst          int   `json:"burst"`
}

// messageStatsOutput is a queue's message counts as of SampledAt and its
// cumulative totals since StartedAt. Both times are absent on the all-zero
// stats a create returns.
type messageStatsOutput struct {
	SampledAt      time.Time `json:"sampledAt,omitzero" human:"age"`
	StartedAt      time.Time `json:"startedAt,omitzero" human:"age"`
	Total          int64     `json:"total"`
	DataBytes      int64     `json:"dataBytes" human:"bytes"`
	Delayed        int64     `json:"delayed"`
	Visible        int64     `json:"visible"`
	Pending        int64     `json:"pending"`
	Failed         int64     `json:"failed"`
	Unavailable    int64     `json:"unavailable"`
	AddedTotal     int64     `json:"addedTotal"`
	AckedTotal     int64     `json:"ackedTotal"`
	NackedTotal    int64     `json:"nackedTotal"`
	ReleasedTotal  int64     `json:"releasedTotal"`
	FailedTotal    int64     `json:"failedTotal"`
	DiscardedTotal int64     `json:"discardedTotal"`
	ExpiredTotal   int64     `json:"expiredTotal"`
}

// formatQueue converts an SDK queue into its output form.
func formatQueue(q spooler.Queue) queueOutput {
	return queueOutput{
		Name:      q.Name,
		UID:       q.UID,
		CreatedAt: q.CreatedAt,
		State:     q.State.String(),
		Settings:  formatQueueSettings(q.Settings),
		Stats:     formatMessageStats(q.Stats),
	}
}

func formatQueueSettings(s spooler.QueueSettings) queueSettingsOutput {
	return queueSettingsOutput{
		LeaseTimeoutSeconds:       s.LeaseTimeoutSeconds,
		MaxRetries:                s.MaxRetries,
		DedupWindowSeconds:        s.DedupWindowSeconds,
		RetentionSeconds:          s.RetentionSeconds,
		EffectiveRetentionSeconds: s.EffectiveRetentionSeconds,
		RecvRateLimit:             formatRateLimit(s.RecvRateLimit),
	}
}

func formatRateLimit(r spooler.RateLimit) rateLimitOutput {
	return rateLimitOutput{
		IntervalMicros: r.IntervalMicros,
		Burst:          r.Burst,
	}
}

func formatMessageStats(s spooler.MessageStats) messageStatsOutput {
	return messageStatsOutput{
		SampledAt:      s.SampledAt,
		StartedAt:      s.StartedAt,
		Total:          s.Total,
		DataBytes:      s.DataBytes,
		Delayed:        s.Delayed,
		Visible:        s.Visible,
		Pending:        s.Pending,
		Failed:         s.Failed,
		Unavailable:    s.Unavailable,
		AddedTotal:     s.AddedTotal,
		AckedTotal:     s.AckedTotal,
		NackedTotal:    s.NackedTotal,
		ReleasedTotal:  s.ReleasedTotal,
		FailedTotal:    s.FailedTotal,
		DiscardedTotal: s.DiscardedTotal,
		ExpiredTotal:   s.ExpiredTotal,
	}
}

// spoolOutput is a spool as the CLI prints it, with the docs' field names
// (https://docs.spooler.sh/operations/spool): its queue count and its
// message counts summed across every queue.
type spoolOutput struct {
	Name       string             `json:"name"`
	QueueCount int64              `json:"queueCount"`
	Stats      messageStatsOutput `json:"stats"`
}

// formatSpool converts an SDK spool into its output form.
func formatSpool(s spooler.Spool) spoolOutput {
	return spoolOutput{
		Name:       s.Name,
		QueueCount: s.QueueCount,
		Stats:      formatMessageStats(s.Stats),
	}
}

// messageOutput is a message as a listing describes it, without its payload,
// with the docs' field names
// (https://docs.spooler.sh/operations/queueMessages).
type messageOutput struct {
	ID             string    `json:"id"`
	State          string    `json:"state"`
	Retries        int       `json:"retries"`
	CreatedAt      time.Time `json:"createdAt" human:"age"`
	VisibleAt      time.Time `json:"visibleAt,omitzero" human:"age"`
	LeaseExpiresAt time.Time `json:"leaseExpiresAt,omitzero" human:"age"`
	ExpiresAt      time.Time `json:"expiresAt,omitzero" human:"age"`
	Handle         string    `json:"handle,omitempty"`
}

// formatMessage converts a listed message into its output form.
func formatMessage(m spooler.Message) messageOutput {
	return messageOutput{
		ID:             m.ID,
		State:          m.State.String(),
		Retries:        m.Retries,
		CreatedAt:      m.CreatedAt,
		VisibleAt:      m.VisibleAt,
		LeaseExpiresAt: m.LeaseExpiresAt,
		ExpiresAt:      m.ExpiresAt,
		Handle:         string(m.Handle),
	}
}

// sendResultOutput is a send's outcome as the CLI prints it: the Spooler-*
// headers of the response by their docs' names
// (https://docs.spooler.sh/operations/sendMessage).
type sendResultOutput struct {
	ID        string `json:"id"`
	Duplicate bool   `json:"duplicate"`
	Handle    string `json:"handle,omitempty"`
}

// formatSendResult converts a send's result into its output form.
func formatSendResult(r spooler.SendResult) sendResultOutput {
	return sendResultOutput{
		ID:        r.ID,
		Duplicate: r.Duplicate,
		Handle:    string(r.Handle),
	}
}

// recvResultOutput is a receive's outcome as the CLI prints it and as
// --lease-file stores it: the Spooler-* headers of the response by their
// docs' names (https://docs.spooler.sh/operations/recvMessage), without the
// payload.
type recvResultOutput struct {
	ID             string    `json:"id"`
	Retries        int       `json:"retries"`
	CreatedAt      time.Time `json:"createdAt" human:"age"`
	Lease          string    `json:"lease"`
	LeaseExpiresAt time.Time `json:"leaseExpiresAt,omitzero" human:"age"`
	DataSize       byteCount `json:"dataSize,omitzero" human:"bytes"`
}

// formatRecvResult converts a receive's result into its output form.
func formatRecvResult(r spooler.RecvResult) recvResultOutput {
	return recvResultOutput{
		ID:             r.ID,
		Retries:        r.Retries,
		CreatedAt:      r.CreatedAt,
		Lease:          string(r.Lease),
		LeaseExpiresAt: r.LeaseExpiresAt,
		DataSize:       byteCount(r.DataSize),
	}
}

// inspectResultOutput is an inspect's outcome as the CLI prints it: the
// Spooler-* headers of the response by their docs' names
// (https://docs.spooler.sh/operations/inspectMessage), the attributes a
// listing reports less the handle, which is the input, without the
// payload.
type inspectResultOutput struct {
	ID             string    `json:"id"`
	State          string    `json:"state"`
	Retries        int       `json:"retries"`
	CreatedAt      time.Time `json:"createdAt" human:"age"`
	VisibleAt      time.Time `json:"visibleAt,omitzero" human:"age"`
	LeaseExpiresAt time.Time `json:"leaseExpiresAt,omitzero" human:"age"`
	ExpiresAt      time.Time `json:"expiresAt,omitzero" human:"age"`
	DataSize       byteCount `json:"dataSize,omitzero" human:"bytes"`
}

// formatInspectResult converts an inspect's result into its output form.
func formatInspectResult(r spooler.InspectResult) inspectResultOutput {
	return inspectResultOutput{
		ID:             r.ID,
		State:          r.State.String(),
		Retries:        r.Retries,
		CreatedAt:      r.CreatedAt,
		VisibleAt:      r.VisibleAt,
		LeaseExpiresAt: r.LeaseExpiresAt,
		ExpiresAt:      r.ExpiresAt,
		DataSize:       byteCount(r.DataSize),
	}
}

// renewResultOutput is a renew's outcome as the CLI prints it: the lease's
// new expiry under its docs' name
// (https://docs.spooler.sh/operations/renewLease), absent when the lease
// never expires. The lease itself is unchanged and not repeated.
type renewResultOutput struct {
	LeaseExpiresAt time.Time `json:"leaseExpiresAt,omitzero" human:"age"`
}

// formatRenewResult converts a renew's result into its output form.
func formatRenewResult(r spooler.RenewResult) renewResultOutput {
	return renewResultOutput{
		LeaseExpiresAt: r.LeaseExpiresAt,
	}
}

// versionOutput is what version prints: the CLI's release and the SDK's.
type versionOutput struct {
	CLI string `json:"cli"`
	SDK string `json:"sdk"`
}

// byteCount is a size the server may not have declared: negative is
// unknown, and is omitted as a zero value would be.
type byteCount int64

// IsZero reports an undeclared size, for omitzero.
func (c byteCount) IsZero() bool {
	return c < 0
}
