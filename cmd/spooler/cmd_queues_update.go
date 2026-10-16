package main

import (
	"context"
	_ "embed"
	"flag"
	"text/template"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-cli/internal/flagvars"
	"github.com/spoolersh/spooler-go"
)

type queuesUpdateCommand struct {
	config *config

	spool      string
	queue      string
	queueGiven bool
	uid        *string
	file       *string
	settings   queueSettingsFlags
}

func (c *queuesUpdateCommand) Summary() string {
	return "Change a queue's settings"
}

func (c *queuesUpdateCommand) Synopsis() string {
	return "--spool <spool> (--queue <queue> | --file <file>) [<options>]"
}

//go:embed cmd_queues_update.txt
var queuesUpdateDescText string

var queuesUpdateDesc *template.Template

func init() {
	queuesUpdateDesc = must(template.New("").Parse(queuesUpdateDescText))
}

func (c *queuesUpdateCommand) Description() string {
	return renderDesc(queuesUpdateDesc)
}

func (c *queuesUpdateCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	fs.Var(flagvars.Opt(flagvars.StringVar(&c.queue), &c.queueGiven),
		"queue",
		"queue to update; omitted means the --file's name",
	)
	defineUIDFlag(&c.uid, fs)
	defineFileFlag(&c.file, fs)
	c.settings.define(fs)
}

func (c *queuesUpdateCommand) Run(ctx context.Context, args []string) error {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	if noFlags(ctx) {
		return new(cli.UsageError)
	}
	var rec queueRecord
	if c.file != nil {
		var err error
		rec, err = readQueueRecord(*c.file)
		if err != nil {
			return err
		}
	}
	// The flag, else the record's name, empty included.
	var name string
	switch {
	case c.queueGiven:
		name = c.queue
	case rec.Name != nil:
		name = *rec.Name
	default:
		return cli.Exitf(2, "no queue: pass --queue or --file")
	}
	settings, err := c.settings.toQueueSettingsUpdate(rec.Settings)
	if err != nil {
		return err
	}
	return c.config.client.UpdateQueue(ctx, spooler.UpdateQueueRequest{
		Spool:    c.spool,
		Queue:    name,
		QueueUID: c.uid,
		Settings: settings,
	})
}
