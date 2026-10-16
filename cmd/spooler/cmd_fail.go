package main

import (
	"context"
	_ "embed"
	"text/template"

	"github.com/spoolersh/spooler-go"
)

type failCommand struct {
	_leaseOpCommand
}

func (c *failCommand) Summary() string {
	return "Fail a leased message"
}

func (c *failCommand) Synopsis() string {
	return "--spool <spool> (--lease <lease> | --lease-file <file>) [<options>]"
}

//go:embed cmd_fail.txt
var failDescText string

var failDesc *template.Template

func init() {
	failDesc = must(template.New("").Parse(failDescText))
}

func (c *failCommand) Description() string {
	return renderDesc(failDesc)
}

func (c *failCommand) Run(ctx context.Context, args []string) error {
	return c.run(ctx, args, func(spool string, lease spooler.Lease) error {
		return c.config.client.Fail(ctx, spooler.FailRequest{
			Spool: spool,
			Lease: lease,
		})
	})
}
