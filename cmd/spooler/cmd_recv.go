package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"text/template"
	"time"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-cli/internal/flagvars"
	"github.com/spoolersh/spooler-go"
)

type recvCommand struct {
	config *config

	spool       string
	queue       string
	waitSeconds *int

	dataFile    *string
	leaseFile   *string
	includeMeta bool
	overwrite   bool
	ack         bool

	output output[recvResultOutput]
}

func (c *recvCommand) Summary() string {
	return "Receive a message, leasing it"
}

func (c *recvCommand) Synopsis() string {
	return "--spool <spool> --queue <queue> (--ack | --lease-file <file> | -i)" +
		" [<options>]"
}

//go:embed cmd_recv.txt
var recvDescText string

var recvDesc *template.Template

func init() {
	recvDesc = must(template.New("").Parse(recvDescText))
}

func (c *recvCommand) Description() string {
	return renderDesc(recvDesc)
}

func (c *recvCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	defineQueueFlag(&c.queue, fs)
	fs.Var(flagvars.OptVar(&c.waitSeconds, parseWholeSeconds),
		"wait",
		"`duration` to wait for a message (20s); omitted means the server's"+
			" default",
	)
	defineDataFileFlag(&c.dataFile, fs)
	fs.Var(flagvars.OptVar(&c.leaseFile, flagvars.Ident),
		"lease-file",
		"file to write the lease to, as JSON",
	)
	defineIncludeMetaFlag(&c.includeMeta, fs)
	defineOverwriteFlag(&c.overwrite, fs)
	fs.BoolVar(&c.ack,
		"ack", false,
		"acknowledge the message once the payload is written",
	)
	c.output.defineFlags(fs)
}

func (c *recvCommand) Run(ctx context.Context, args []string) (err error) {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	if noFlags(ctx) {
		return new(cli.UsageError)
	}
	// A receive leases the message, so the caller says what becomes of the
	// lease; an omitted flag must not lease silently.
	if !c.ack && c.leaseFile == nil && !c.includeMeta {
		return cli.Exitf(2,
			"a receive leases the message: pass --ack to settle it,"+
				" --lease-file to keep the lease,"+
				" or --include-meta to print it",
		)
	}
	// Only --include-meta renders through --output; a lease file is JSON.
	if !c.includeMeta {
		if err := c.output.unusedError("--include-meta"); err != nil {
			return err
		}
	}

	// Every destination opens before the receive, so a refusal costs no
	// lease, and nothing received leaves the files untouched. The lease file
	// opens first: when one path names both files, the lease's mode wins.
	fm := fileman{
		overwrite: c.overwrite,
	}
	defer fm.Cleanup(&err)

	var leaseFile *os.File
	if c.leaseFile != nil {
		leaseFile, err = fm.Open(*c.leaseFile, 0600)
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
	leaseShared, err := sameFile(leaseFile, dataFile)
	if err != nil {
		return err
	}
	metaShared, err := sameFile(os.Stdout, dataFile)
	if err != nil {
		return err
	}
	if c.includeMeta {
		both, err := sameFile(leaseFile, os.Stdout)
		if err != nil {
			return err
		}
		if both {
			return cli.Exitf(2,
				"--include-meta and --lease-file %s both print the lease"+
					" on stdout; pass one",
				*c.leaseFile,
			)
		}
	}
	// The output flags are checked here, before the receive, as the files
	// are: a usage error must not cost a lease.
	var metaEnc *metaEncoder[recvResultOutput]
	if c.includeMeta {
		metaEnc, err = newMetaEncoder(os.Stdout, &c.output)
		if err != nil {
			return err
		}
	}

	req := spooler.RecvRequest{
		Spool:       c.spool,
		Queue:       c.queue,
		WaitSeconds: c.waitSeconds,
		Stream:      true,
	}
	res, ok, err := c.config.client.Recv(ctx, req)
	if err != nil {
		return err
	}
	if !ok {
		// A code alone: nothing received is exit 1 in silence, as grep on
		// no match.
		return &cli.ExitError{
			Code: 1,
		}
	}
	defer res.DataStream.Close()

	// From here on the message is leased: a failure to deliver it releases
	// the lease, so the message is visible again at once. With --ack alone
	// there is no record: the lease is settled, nothing is kept.
	meta := formatRecvResult(res)
	if c.includeMeta {
		err = metaEnc.encode(meta, metaShared)
	}
	if err == nil {
		err = writeMeta(leaseFile, meta, leaseShared)
	}
	if err != nil {
		// A half-written record is not worth keeping: the lease is in the
		// message when the release fails.
		return c.release(ctx, &fm, nil, res.Lease, fmt.Errorf(
			"write lease: %w",
			err,
		))
	}
	if _, err := io.Copy(dataFile, res.DataStream); err != nil {
		return c.release(ctx, &fm, leaseFile, res.Lease, fmt.Errorf(
			"write payload: %w",
			err,
		))
	}
	// A close error is a write error for the payload's last bytes.
	if err := fm.Close(dataFile); err != nil {
		return c.release(ctx, &fm, leaseFile, res.Lease, fmt.Errorf(
			"write payload: %w",
			err,
		))
	}
	// The lease file closes last, so a record replaces an existing file only
	// once the payload is delivered: a delivery that fails releases the
	// lease, and the old record is the one worth keeping then. Closed, the
	// files are the caller's, with the lease in them, whatever the ack says.
	if err := fm.Close(leaseFile); err != nil {
		return c.release(ctx, &fm, nil, res.Lease, fmt.Errorf(
			"write lease: %w",
			err,
		))
	}
	if c.ack {
		err := c.config.client.Ack(ctx, spooler.AckRequest{
			Spool: c.spool,
			Lease: res.Lease,
		})
		if errors.Is(err, spooler.ErrResultUnknown) {
			return fmt.Errorf(
				"ack: %w; whether the message is settled or still leased is"+
					" unknown (lease %q)",
				err, string(res.Lease),
			)
		}
		if err != nil {
			// Nothing about the lease's state: a stale token may be an expired
			// lease or a settled one, and the SDK's error says which. Coded
			// here, since an error naming a lease is reported past a signal.
			return cli.Exitf(exitCode(err),
				"ack: %w (lease %q)",
				err, string(res.Lease),
			)
		}
	}
	return nil
}

// release gives the lease back after cause, so the message is visible again
// at once, and reports both outcomes as one error. It runs even when ctx is
// done, since that is when it is needed most. When the release fails or ends
// unknown the lease may be live, so keep, the lease file holding it, is
// closed as delivered instead of removed; nil is no file to keep.
func (c *recvCommand) release(ctx context.Context, fm *fileman, keep *os.File, lease spooler.Lease, cause error) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	err := c.config.client.Release(ctx, spooler.ReleaseRequest{
		Spool: c.spool,
		Lease: lease,
	})
	if err == nil {
		return fmt.Errorf(
			"%w; the lease was released",
			cause,
		)
	}
	// Its error goes unreported: the lease is in the message either way.
	fm.Close(keep)
	if errors.Is(err, spooler.ErrResultUnknown) {
		return fmt.Errorf(
			"%w; release: %w; whether the message is released or still"+
				" leased is unknown (lease %q)",
			cause, err, string(lease),
		)
	}
	// The delivery's own code, set here since an error naming a lease is
	// reported past a signal.
	return cli.Exitf(exitCode(cause),
		"%w; release: %v (lease %q)",
		cause, err, string(lease),
	)
}
