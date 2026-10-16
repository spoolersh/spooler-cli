package main

import (
	"context"
	_ "embed"
	"text/template"

	"github.com/spoolersh/spooler-go"
)

type nackCommand struct {
	_leaseOpCommand
}

func (c *nackCommand) Summary() string {
	return "Return a leased message for redelivery"
}

func (c *nackCommand) Synopsis() string {
	return "--spool <spool> (--lease <lease> | --lease-file <file>) [<options>]"
}

//go:embed cmd_nack.txt
var nackDescText string

var nackDesc *template.Template

func init() {
	nackDesc = must(template.New("").Parse(nackDescText))
}

func (c *nackCommand) Description() string {
	return renderDesc(nackDesc)
}

func (c *nackCommand) Run(ctx context.Context, args []string) error {
	return c.run(ctx, args, func(spool string, lease spooler.Lease) error {
		return c.config.client.Nack(ctx, spooler.NackRequest{
			Spool: spool,
			Lease: lease,
		})
	})
}
