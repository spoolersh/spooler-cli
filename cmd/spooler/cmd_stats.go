package main

import (
	"context"
	_ "embed"
	"flag"
	"os"
	"text/template"

	"github.com/spoolersh/spooler-go"
)

type statsCommand struct {
	config *config

	spool string

	output output[spoolOutput]
}

func (c *statsCommand) Summary() string {
	return "Describe the spool"
}

func (c *statsCommand) Synopsis() string {
	return "--spool <spool> [<options>]"
}

//go:embed cmd_stats.txt
var statsDescText string

var statsDesc *template.Template

func init() {
	statsDesc = must(template.New("").Parse(statsDescText))
}

func (c *statsCommand) Description() string {
	return renderDesc(statsDesc)
}

func (c *statsCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	c.output.defineFlags(fs)
}

func (c *statsCommand) Run(ctx context.Context, args []string) error {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	enc, err := c.output.encoderTo(os.Stdout)
	if err != nil {
		return err
	}
	spool, err := c.config.client.Spool(ctx, spooler.SpoolRequest{
		Spool: c.spool,
	})
	if err != nil {
		return err
	}
	if err := enc.Encode(formatSpool(spool)); err != nil {
		return err
	}
	return enc.Flush()
}
