package main

import (
	"context"
	_ "embed"
	"flag"
	"os"
	"text/template"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-go"
)

type queuesDescribeCommand struct {
	config *config

	spool string
	queue string

	output output[queueOutput]
}

func (c *queuesDescribeCommand) Summary() string {
	return "Describe one queue"
}

func (c *queuesDescribeCommand) Synopsis() string {
	return "--spool <spool> --queue <queue> [<options>]"
}

//go:embed cmd_queues_describe.txt
var queuesDescribeDescText string

var queuesDescribeDesc *template.Template

func init() {
	queuesDescribeDesc = must(template.New("").Parse(queuesDescribeDescText))
}

func (c *queuesDescribeCommand) Description() string {
	return renderDesc(queuesDescribeDesc)
}

func (c *queuesDescribeCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	defineQueueFlag(&c.queue, fs)
	c.output.defineFlags(fs)
}

func (c *queuesDescribeCommand) Run(ctx context.Context, args []string) error {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	if noFlags(ctx) {
		return new(cli.UsageError)
	}
	enc, err := c.output.encoderTo(os.Stdout)
	if err != nil {
		return err
	}
	queue, err := c.config.client.Queue(ctx, spooler.QueueRequest{
		Spool: c.spool,
		Queue: c.queue,
	})
	if err != nil {
		return err
	}
	if err := enc.Encode(formatQueue(queue)); err != nil {
		return err
	}
	return enc.Flush()
}
