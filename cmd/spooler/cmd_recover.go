package main

import (
	"context"
	_ "embed"
	"flag"
	"text/template"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-go"
)

type recoverCommand struct {
	config *config

	spool        string
	handle       *string
	delaySeconds *int
}

func (c *recoverCommand) Summary() string {
	return "Return a failed message to its queue"
}

func (c *recoverCommand) Synopsis() string {
	return "--spool <spool> --handle <handle> [<options>]"
}

//go:embed cmd_recover.txt
var recoverDescText string

var recoverDesc *template.Template

func init() {
	recoverDesc = must(template.New("").Parse(recoverDescText))
}

func (c *recoverCommand) Description() string {
	return renderDesc(recoverDesc)
}

func (c *recoverCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	defineHandleFlag(&c.handle, fs)
	defineDelayFlag(&c.delaySeconds, fs)
}

func (c *recoverCommand) Run(ctx context.Context, args []string) error {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	if noFlags(ctx) {
		return new(cli.UsageError)
	}
	if c.handle == nil {
		return cli.Exitf(2, "no message: pass --handle")
	}
	req := spooler.RecoverRequest{
		Spool:  c.spool,
		Handle: spooler.Handle(*c.handle),
	}
	if c.delaySeconds != nil {
		req.DelaySeconds = *c.delaySeconds
	}
	return c.config.client.Recover(ctx, req)
}
