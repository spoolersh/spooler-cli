package main

import (
	"context"
	_ "embed"
	"flag"
	"text/template"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-go"
)

type discardCommand struct {
	config *config

	spool     string
	handle    *string
	lease     *string
	leaseFile *string
	yes       bool
}

func (c *discardCommand) Summary() string {
	return "Drop a message for good"
}

func (c *discardCommand) Synopsis() string {
	return "--spool <spool> (--handle <handle> | --lease <lease> |" +
		" --lease-file <file>) --yes [<options>]"
}

//go:embed cmd_discard.txt
var discardDescText string

var discardDesc *template.Template

func init() {
	discardDesc = must(template.New("").Parse(discardDescText))
}

func (c *discardCommand) Description() string {
	return renderDesc(discardDesc)
}

func (c *discardCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	defineHandleFlag(&c.handle, fs)
	defineLeaseFlag(&c.lease, fs)
	defineLeaseFileFlag(&c.leaseFile, fs)
	defineYesFlag(&c.yes, fs)
}

func (c *discardCommand) Run(ctx context.Context, args []string) error {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	if noFlags(ctx) {
		return new(cli.UsageError)
	}
	hasLease := c.lease != nil || c.leaseFile != nil
	switch {
	case c.handle == nil && !hasLease:
		return cli.Exitf(2,
			"no message: pass --handle, --lease or --lease-file",
		)
	case c.handle != nil && hasLease:
		return cli.Exitf(2,
			"--handle and --lease or --lease-file are mutually exclusive",
		)
	}
	req := spooler.DiscardRequest{
		Spool: c.spool,
	}
	if c.handle != nil {
		req.Handle = spooler.Handle(*c.handle)
	}
	if hasLease {
		lease, err := readLease(c.lease, c.leaseFile)
		if err != nil {
			return err
		}
		req.Lease = lease
	}
	if !c.yes {
		return cli.Exitf(2,
			"discard drops the message and its payload for good:"+
				" pass --yes to confirm",
		)
	}
	return c.config.client.Discard(ctx, req)
}
