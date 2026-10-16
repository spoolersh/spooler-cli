package main

import (
	"context"

	"github.com/spoolersh/spooler-go"
)

// client is the SDK as the commands use it: the Client's methods they
// call, as an interface so that tests stub it. The SDK's iterator helpers,
// spooler.Queues and spooler.Messages, take it as the pager they walk.
type client interface {
	Send(context.Context, spooler.SendRequest) (spooler.SendResult, error)
	Recv(context.Context, spooler.RecvRequest) (spooler.RecvResult, bool, error)
	Ack(context.Context, spooler.AckRequest) error
	Nack(context.Context, spooler.NackRequest) error
	Release(context.Context, spooler.ReleaseRequest) error
	Fail(context.Context, spooler.FailRequest) error
	Renew(context.Context, spooler.RenewRequest) (spooler.RenewResult, error)
	Inspect(context.Context, spooler.InspectRequest) (spooler.InspectResult, error)
	Recover(context.Context, spooler.RecoverRequest) error
	Discard(context.Context, spooler.DiscardRequest) error
	Messages(context.Context, spooler.MessagesRequest) (spooler.Page[spooler.Message], error)

	Spool(context.Context, spooler.SpoolRequest) (spooler.Spool, error)
	Queues(context.Context, spooler.QueuesRequest) (spooler.Page[spooler.Queue], error)
	Queue(context.Context, spooler.QueueRequest) (spooler.Queue, error)
	CreateQueue(context.Context, spooler.CreateQueueRequest) (spooler.Queue, error)
	UpdateQueue(context.Context, spooler.UpdateQueueRequest) error
	DeleteQueue(context.Context, spooler.DeleteQueueRequest) error
}

var _ client = (*spooler.Client)(nil)
