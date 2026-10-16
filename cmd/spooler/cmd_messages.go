package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"os"
	"regexp"
	"text/template"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-go"
)

type messagesCommand struct {
	config *config

	spool        string
	queue        string
	state        spooler.MessageState
	match        *regexp.Regexp
	returnHandle bool
	limit        int

	output output[messageOutput]
}

func (c *messagesCommand) Summary() string {
	return "List a queue's messages"
}

func (c *messagesCommand) Synopsis() string {
	return "--spool <spool> --queue <queue> [<options>]"
}

//go:embed cmd_messages.txt
var messagesDescText string

var messagesDesc *template.Template

func init() {
	messagesDesc = must(template.New("").Parse(messagesDescText))
}

func (c *messagesCommand) Description() string {
	return renderDesc(messagesDesc)
}

func (c *messagesCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	defineQueueFlag(&c.queue, fs)
	defineMessageStateFlag(&c.state, spooler.MessageStateUnknown, fs)
	defineReturnHandleFlag(&c.returnHandle, false, fs)
	defineLimitFlag(&c.limit, fs)
	c.output.defineFlags(fs)
}

func (c *messagesCommand) Run(ctx context.Context, args []string) (err error) {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	if noFlags(ctx) {
		return new(cli.UsageError)
	}
	// The default columns are every attribute; the ones the flags leave
	// empty on every row go.
	var empty []string
	if !c.returnHandle {
		empty = append(empty, "handle")
	}
	switch c.state {
	case spooler.MessageStateUnknown:
		// Every state: any column may be filled.

	case spooler.MessageStateDelayed:
		empty = append(empty, "leaseExpiresAt")

	case spooler.MessageStatePending:
		empty = append(empty, "visibleAt")

	default:
		empty = append(empty, "visibleAt", "leaseExpiresAt")
	}
	if err := c.output.removeDefaultColumns(empty...); err != nil {
		panic(err)
	}
	enc, err := c.output.encoderTo(os.Stdout)
	if err != nil {
		return err
	}
	messages := list(c.limit, func(after string) (spooler.Page[spooler.Message], error) {
		return c.config.client.Messages(ctx, spooler.MessagesRequest{
			Spool:        c.spool,
			Queue:        c.queue,
			State:        c.state,
			ReturnHandle: c.returnHandle,
			After:        after,
		})
	})
	for message, err := range messages {
		if err != nil {
			// What was fetched is printed whole before the error says the
			// listing is cut, as the json stream already does; a reader
			// has the exit code to know it is partial.
			return errors.Join(err, enc.Flush())
		}
		if err := enc.Encode(formatMessage(message)); err != nil {
			return err
		}
	}
	return enc.Flush()
}
