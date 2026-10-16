package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-cli/internal/flagvars"
	"github.com/spoolersh/spooler-go"
)

type sendCommand struct {
	config *config

	spool        string
	queue        string
	delaySeconds *int
	dedupHash    *string
	dedupString  *string
	returnHandle bool

	data     *string
	dataFile *string

	output output[sendResultOutput]
}

func (c *sendCommand) Summary() string {
	return "Send a message to a queue"
}

func (c *sendCommand) Synopsis() string {
	return "--spool <spool> --queue <queue> (--data <data> | --data-file <file>)" +
		" [<options>]"
}

//go:embed cmd_send.txt
var sendDescText string

var sendDesc *template.Template

func init() {
	sendDesc = must(template.New("").Parse(sendDescText))
}

func (c *sendCommand) Description() string {
	return renderDesc(sendDesc)
}

func (c *sendCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	defineQueueFlag(&c.queue, fs)
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

func (c *sendCommand) Run(ctx context.Context, args []string) (err error) {
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
	var dataSource spooler.DataSource
	switch {
	case c.data != nil && c.dataFile != nil:
		return cli.Exitf(2, "--data and --data-file are mutually exclusive")

	case c.data == nil && c.dataFile == nil:
		return cli.Exitf(2,
			"no payload: pass --data, --data-file, or `--data-file -` for stdin",
		)

	case c.data != nil:
		dataSource = spooler.DataSourceReadSeeker(strings.NewReader(*c.data))

	case *c.dataFile == "-":
		// NOTE: stdin is not seekable, so no retries are possible.
		dataSource = spooler.DataSourceReader(os.Stdin)

	default:
		file, err := os.Open(*c.dataFile)
		if err != nil {
			return err
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			// A regular file can be rewound, which enables retries.
			dataSource = spooler.DataSourceReadSeeker(file)
		} else {
			// A pipe, FIFO or device (<(cmd), /dev/stdin) can't seek: the
			// SDK would fail measuring it, so stream it once.
			dataSource = spooler.DataSourceReader(file)
		}
	}

	enc, err := c.output.encoderTo(os.Stdout)
	if err != nil {
		return err
	}

	req := spooler.SendRequest{
		Spool:        c.spool,
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
	res, err := c.config.client.Send(ctx, req)
	if errors.Is(err, spooler.ErrResultUnknown) {
		// The one case where a blind retry can duplicate. A key protects
		// only a retry of a send that carried it from the first attempt;
		// one added afterwards protects nothing.
		if c.dedupHash == nil && c.dedupString == nil {
			return fmt.Errorf(
				"%w; the message may have been sent, and a retry may send"+
					" it twice: only a send whose first attempt carried"+
					" --dedup-string or --dedup-hash is safe to retry, with"+
					" the same value, within the queue's dedup window",
				err,
			)
		}
		return fmt.Errorf(
			"%w; a retry with the same --dedup-string or --dedup-hash is"+
				" safe within the queue's dedup window",
			err,
		)
	}
	if err != nil {
		return err
	}
	if err := enc.Encode(formatSendResult(res)); err != nil {
		return err
	}
	return enc.Flush()
}
