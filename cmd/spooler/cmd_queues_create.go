package main

import (
	"context"
	_ "embed"
	"flag"
	"os"
	"text/template"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-cli/internal/flagvars"
	"github.com/spoolersh/spooler-go"
)

type queuesCreateCommand struct {
	config *config

	spool      string
	queue      string
	queueGiven bool
	file       *string
	settings   queueSettingsFlags

	output output[queueOutput]
}

func (c *queuesCreateCommand) Summary() string {
	return "Create a queue"
}

func (c *queuesCreateCommand) Synopsis() string {
	return "--spool <spool> (--queue <queue> | --file <file>) [<options>]"
}

//go:embed cmd_queues_create.txt
var queuesCreateDescText string

var queuesCreateDesc *template.Template

func init() {
	queuesCreateDesc = must(template.New("").Parse(queuesCreateDescText))
}

func (c *queuesCreateCommand) Description() string {
	return renderDesc(queuesCreateDesc)
}

func (c *queuesCreateCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	fs.Var(flagvars.Opt(flagvars.StringVar(&c.queue), &c.queueGiven),
		"queue",
		"queue to create; omitted means the --file's name",
	)
	defineFileFlag(&c.file, fs)
	c.settings.define(fs)
	c.output.defineFlags(fs)
}

func (c *queuesCreateCommand) Run(ctx context.Context, args []string) error {
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
	settings, err := c.settings.toQueueSettingsCreate(rec.Settings)
	if err != nil {
		return err
	}
	enc, err := c.output.encoderTo(os.Stdout)
	if err != nil {
		return err
	}
	queue, err := c.config.client.CreateQueue(ctx, spooler.CreateQueueRequest{
		Spool:    c.spool,
		Queue:    name,
		Settings: settings,
	})
	if err != nil {
		return err
	}
	if err := enc.Encode(formatQueue(queue)); err != nil {
		return err
	}
	return enc.Flush()
}
