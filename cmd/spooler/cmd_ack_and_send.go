package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"text/template"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-cli/internal/flagvars"
	"github.com/spoolersh/spooler-go"
)

type ackAndSendCommand struct {
	config *config

	spool        string
	queue        string
	lease        *string
	leaseFile    *string
	delaySeconds *int
	dedupHash    *string
	dedupString  *string
	returnHandle bool

	data     *string
	dataFile *string

	output output[ackAndSendResultOutput]
}

func (c *ackAndSendCommand) Summary() string {
	return "Acknowledge a leased message and send another, atomically"
}

func (c *ackAndSendCommand) Synopsis() string {
	return "--spool <spool> (--lease <lease> | --lease-file <file>)" +
		" --queue <queue> (--data <data> | --data-file <file>) [<options>]"
}

//go:embed cmd_ack_and_send.txt
var ackAndSendDescText string

var ackAndSendDesc *template.Template

func init() {
	ackAndSendDesc = must(template.New("").Parse(ackAndSendDescText))
}

func (c *ackAndSendCommand) Description() string {
	return renderDesc(ackAndSendDesc)
}

func (c *ackAndSendCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	defineQueueFlag(&c.queue, fs)
	defineLeaseFlag(&c.lease, fs)
	defineLeaseFileFlag(&c.leaseFile, fs)
	defineDelayFlag(&c.delaySeconds, fs)
	fs.Var(flagvars.OptVar(&c.data, flagvars.Ident),
		"data",
		"payload to send",
	)
	fs.Var(flagvars.OptVar(&c.dataFile, flagvars.Ident),
		"data-file",
		"file to read the payload from; - is stdin",
	)
	fs.Var(flagvars.OptVar(&c.dedupHash, flagvars.Ident),
		"dedup-hash",
		"dedup hash",
	)
	fs.Var(flagvars.OptVar(&c.dedupString, flagvars.Ident),
		"dedup-string",
		"dedup string",
	)
	defineReturnHandleFlag(&c.returnHandle, false, fs)
	c.output.defineFlags(fs)
}

func (c *ackAndSendCommand) Run(ctx context.Context, args []string) (err error) {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	if noFlags(ctx) {
		return new(cli.UsageError)
	}
	if c.dedupHash != nil && c.dedupString != nil {
		return cli.Exitf(2,
			"--dedup-hash and --dedup-string are mutually exclusive",
		)
	}
	lease, err := readLease(c.lease, c.leaseFile)
	if err != nil {
		return err
	}
	dataSource, closer, err := readData(c.data, c.dataFile)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}

	enc, err := c.output.encoderTo(os.Stdout)
	if err != nil {
		return err
	}

	req := spooler.AckAndSendRequest{
		Spool:        c.spool,
		Lease:        lease,
		Queue:        c.queue,
		DataSource:   dataSource,
		ReturnHandle: c.returnHandle,
	}
	if c.delaySeconds != nil {
		req.DelaySeconds = *c.delaySeconds
	}
	if c.dedupHash != nil {
		req.Dedup = spooler.DedupHash(*c.dedupHash)
	}
	if c.dedupString != nil {
		req.Dedup = spooler.DedupString(*c.dedupString)
	}
	res, err := c.config.client.AckAndSend(ctx, req)
	if errors.Is(err, spooler.ErrResultUnknown) {
		// Unlike send, a retry needs no dedup key: the docs rule that a
		// retry of a call that committed is refused and sends nothing.
		return fmt.Errorf(
			"%w; a retry is safe: a retry of a call that committed is"+
				" refused as stale and sends nothing",
			err,
		)
	}
	if err != nil {
		return err
	}
	if err := enc.Encode(formatAckAndSendResult(res)); err != nil {
		return err
	}
	return enc.Flush()
}
