package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-cli/internal/flagvars"
	"github.com/spoolersh/spooler-go"
)

type inspectCommand struct {
	config *config

	spool  string
	handle *string

	dataFile    *string
	metaFile    *string
	includeMeta bool
	overwrite   bool

	output output[inspectResultOutput]
}

func (c *inspectCommand) Summary() string {
	return "Read a message by its handle, without leasing it"
}

func (c *inspectCommand) Synopsis() string {
	return "--spool <spool> --handle <handle> [<options>]"
}

//go:embed cmd_inspect.txt
var inspectDescText string

var inspectDesc *template.Template

func init() {
	inspectDesc = must(template.New("").Parse(inspectDescText))
}

func (c *inspectCommand) Description() string {
	return renderDesc(inspectDesc)
}

func (c *inspectCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	defineHandleFlag(&c.handle, fs)
	defineDataFileFlag(&c.dataFile, fs)
	fs.Var(flagvars.OptVar(&c.metaFile, flagvars.Ident),
		"meta-file",
		"file to write the message's attributes to, as JSON",
	)
	defineIncludeMetaFlag(&c.includeMeta, fs)
	defineOverwriteFlag(&c.overwrite, fs)
	c.output.defineFlags(fs)
}

func (c *inspectCommand) Run(ctx context.Context, args []string) (err error) {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	if noFlags(ctx) {
		return new(cli.UsageError)
	}
	if c.handle == nil {
		return cli.Exitf(2, "no message: pass --handle")
	}
	// Only --include-meta renders through --output; a meta file is JSON.
	if !c.includeMeta {
		if err := c.output.unusedError("--include-meta"); err != nil {
			return err
		}
	}

	// Every destination opens before the inspect, so a refusal costs no
	// request, and a failure leaves no file behind. The meta file opens
	// first, as recv opens the lease file first.
	fm := fileman{
		overwrite: c.overwrite,
	}
	defer fm.Cleanup(&err)

	var metaFile *os.File
	if c.metaFile != nil {
		metaFile, err = fm.Open(*c.metaFile, 0666)
		if err != nil {
			return err
		}
	}
	dataFile, err := fm.Open(valueOr(c.dataFile, "-"), 0666)
	if err != nil {
		return err
	}
	// Where a record and the payload share a stream, the record leads and an
	// empty line separates them; two records on one stream would leave a
	// reader no way to tell where the payload starts.
	fileShared, err := sameFile(metaFile, dataFile)
	if err != nil {
		return err
	}
	metaShared, err := sameFile(os.Stdout, dataFile)
	if err != nil {
		return err
	}
	if c.includeMeta {
		both, err := sameFile(metaFile, os.Stdout)
		if err != nil {
			return err
		}
		if both {
			return cli.Exitf(2,
				"--include-meta and --meta-file %s both print the attributes"+
					" on stdout; pass one",
				*c.metaFile,
			)
		}
	}
	// The output flags are checked here, before the request, as the files
	// are.
	var metaEnc *metaEncoder[inspectResultOutput]
	if c.includeMeta {
		metaEnc, err = newMetaEncoder(os.Stdout, &c.output)
		if err != nil {
			return err
		}
	}

	req := spooler.InspectRequest{
		Spool:  c.spool,
		Handle: spooler.Handle(*c.handle),
		Stream: true,
	}
	res, err := c.config.client.Inspect(ctx, req)
	if err != nil {
		return err
	}
	defer res.DataStream.Close()

	meta := formatInspectResult(res)
	if c.includeMeta {
		err = metaEnc.encode(meta, metaShared)
	}
	if err == nil {
		err = writeMeta(metaFile, meta, fileShared)
	}
	if err != nil {
		return fmt.Errorf(
			"write meta: %w",
			err,
		)
	}
	if _, err := io.Copy(dataFile, res.DataStream); err != nil {
		return fmt.Errorf(
			"write payload: %w",
			err,
		)
	}
	// A close error is a write error for the payload's last bytes.
	if err := fm.Close(dataFile); err != nil {
		return fmt.Errorf(
			"write payload: %w",
			err,
		)
	}
	// The meta file closes last, as recv's lease file does: a record
	// replaces an existing file only once the payload is delivered.
	if err := fm.Close(metaFile); err != nil {
		return fmt.Errorf(
			"write meta: %w",
			err,
		)
	}
	return nil
}
