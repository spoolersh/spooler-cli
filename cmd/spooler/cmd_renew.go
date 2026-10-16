package main

import (
	"context"
	_ "embed"
	"flag"
	"os"
	"text/template"

	"github.com/spoolersh/spooler-go"
)

type renewCommand struct {
	_leaseOpCommand

	output output[renewResultOutput]
}

func (c *renewCommand) Summary() string {
	return "Extend a lease"
}

func (c *renewCommand) Synopsis() string {
	return "--spool <spool> (--lease <lease> | --lease-file <file>) [<options>]"
}

//go:embed cmd_renew.txt
var renewDescText string

var renewDesc *template.Template

func init() {
	renewDesc = must(template.New("").Parse(renewDescText))
}

func (c *renewCommand) Description() string {
	return renderDesc(renewDesc)
}

func (c *renewCommand) DefineFlags(fs *flag.FlagSet) {
	c._leaseOpCommand.DefineFlags(fs)
	c.output.defineFlags(fs)
}

func (c *renewCommand) Run(ctx context.Context, args []string) error {
	enc, err := c.output.encoderTo(os.Stdout)
	if err != nil {
		return err
	}
	return c.run(ctx, args, func(spool string, lease spooler.Lease) error {
		res, err := c.config.client.Renew(ctx, spooler.RenewRequest{
			Spool: spool,
			Lease: lease,
		})
		if err != nil {
			return err
		}
		if err := enc.Encode(formatRenewResult(res)); err != nil {
			return err
		}
		return enc.Flush()
	})
}
