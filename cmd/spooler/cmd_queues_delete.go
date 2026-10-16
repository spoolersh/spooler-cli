package main

import (
	"context"
	_ "embed"
	"flag"
	"text/template"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-go"
)

type queuesDeleteCommand struct {
	config *config

	spool string
	queue string
	uid   *string
	mode  spooler.QueueDeleteMode
	yes   bool
}

func (c *queuesDeleteCommand) Summary() string {
	return "Delete a queue"
}

func (c *queuesDeleteCommand) Synopsis() string {
	return "--spool <spool> --queue <queue> --mode <mode> --yes [<options>]"
}

//go:embed cmd_queues_delete.txt
var queuesDeleteDescText string

var queuesDeleteDesc *template.Template

func init() {
	queuesDeleteDesc = must(template.New("").Parse(queuesDeleteDescText))
}

func (c *queuesDeleteCommand) Description() string {
	return renderDesc(queuesDeleteDesc)
}

func (c *queuesDeleteCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	defineQueueFlag(&c.queue, fs)
	defineUIDFlag(&c.uid, fs)
	defineQueueDeleteModeFlag(&c.mode, fs)
	defineYesFlag(&c.yes, fs)
}

func (c *queuesDeleteCommand) Run(ctx context.Context, args []string) error {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	if noFlags(ctx) {
		return new(cli.UsageError)
	}
	if c.mode == spooler.QueueDeleteModeUnknown {
		return cli.Exitf(2,
			"no mode: pass --mode force to delete the queue and its messages"+
				" at once, or --mode drain to delete it once consumers have"+
				" emptied it",
		)
	}
	if !c.yes {
		return cli.Exitf(2,
			"queues delete removes the queue %q for good:"+
				" pass --yes to confirm",
			c.queue,
		)
	}
	return c.config.client.DeleteQueue(ctx, spooler.DeleteQueueRequest{
		Spool:    c.spool,
		Queue:    c.queue,
		QueueUID: c.uid,
		Mode:     c.mode,
	})
}
