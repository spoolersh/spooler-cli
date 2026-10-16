package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"os"
	"regexp"
	"text/template"

	"github.com/spoolersh/spooler-go"
)

type queuesListCommand struct {
	config *config

	spool string
	match *regexp.Regexp
	limit int

	output output[queueOutput]
}

func (c *queuesListCommand) Summary() string {
	return "List the spool's queues"
}

func (c *queuesListCommand) Synopsis() string {
	return "--spool <spool> [<options>]"
}

//go:embed cmd_queues_list.txt
var queuesListDescText string

var queuesListDesc *template.Template

func init() {
	queuesListDesc = must(template.New("").Parse(queuesListDescText))
}

func (c *queuesListCommand) Description() string {
	return renderDesc(queuesListDesc)
}

func (c *queuesListCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	defineMatchFlag(&c.match, fs)
	defineLimitFlag(&c.limit, fs)
	c.output.defineFlags(fs)
}

func (c *queuesListCommand) Run(ctx context.Context, args []string) error {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	enc, err := c.output.encoderTo(os.Stdout)
	if err != nil {
		return err
	}
	queues := list(c.limit, func(after string) (spooler.Page[spooler.Queue], error) {
		return c.config.client.Queues(ctx, spooler.QueuesRequest{
			Spool: c.spool,
			Match: c.match,
			After: after,
		})
	})
	for queue, err := range queues {
		if err != nil {
			// What was fetched is printed whole before the error says the
			// listing is cut, as the json stream already does; a reader
			// has the exit code to know it is partial.
			return errors.Join(err, enc.Flush())
		}
		if err := enc.Encode(formatQueue(queue)); err != nil {
			return err
		}
	}
	return enc.Flush()
}
