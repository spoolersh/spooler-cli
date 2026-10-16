package main

import (
	"context"
	_ "embed"
	"flag"
	"text/template"

	"github.com/spoolersh/spooler-go"
)

type releaseCommand struct {
	_leaseOpCommand

	delaySeconds *int
}

func (c *releaseCommand) Summary() string {
	return "Return a leased message without counting a retry"
}

func (c *releaseCommand) Synopsis() string {
	return "--spool <spool> (--lease <lease> | --lease-file <file>) [<options>]"
}

//go:embed cmd_release.txt
var releaseDescText string

var releaseDesc *template.Template

func init() {
	releaseDesc = must(template.New("").Parse(releaseDescText))
}

func (c *releaseCommand) Description() string {
	return renderDesc(releaseDesc)
}

func (c *releaseCommand) DefineFlags(fs *flag.FlagSet) {
	c._leaseOpCommand.DefineFlags(fs)
	defineDelayFlag(&c.delaySeconds, fs)
}

func (c *releaseCommand) Run(ctx context.Context, args []string) error {
	return c.run(ctx, args, func(spool string, lease spooler.Lease) error {
		req := spooler.ReleaseRequest{
			Spool: spool,
			Lease: lease,
		}
		if c.delaySeconds != nil {
			req.DelaySeconds = *c.delaySeconds
		}
		return c.config.client.Release(ctx, req)
	})
}
