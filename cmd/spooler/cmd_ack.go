package main

import (
	"context"
	_ "embed"
	"text/template"

	"github.com/spoolersh/spooler-go"
)

type ackCommand struct {
	_leaseOpCommand
}

func (c *ackCommand) Summary() string {
	return "Acknowledge a leased message, removing it"
}

func (c *ackCommand) Synopsis() string {
	return "--spool <spool> (--lease <lease> | --lease-file <file>) [<options>]"
}

//go:embed cmd_ack.txt
var ackDescText string

var ackDesc *template.Template

func init() {
	ackDesc = must(template.New("").Parse(ackDescText))
}

func (c *ackCommand) Description() string {
	return renderDesc(ackDesc)
}

func (c *ackCommand) Run(ctx context.Context, args []string) error {
	return c.run(ctx, args, func(spool string, lease spooler.Lease) error {
		return c.config.client.Ack(ctx, spooler.AckRequest{
			Spool: spool,
			Lease: lease,
		})
	})
}
