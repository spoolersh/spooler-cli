package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-go"
)

// readLease resolves the lease the settle verbs take: the --lease value, or
// the first JSON object in the --lease-file file, whose lease member is the
// lease and whose other members, and whatever follows the object, are
// ignored, so a file holding the payload too reads the same. "-" is stdin.
// Neither flag or both is a usage error. A nil flag is one omitted; an empty
// value is a value, a lease the API judges or a path the OS does.
func readLease(lease, leaseFile *string) (spooler.Lease, error) {
	switch {
	case lease == nil && leaseFile == nil:
		return "", cli.Exitf(2,
			"no lease: pass --lease or --lease-file",
		)
	case lease != nil && leaseFile != nil:
		return "", cli.Exitf(2,
			"--lease and --lease-file are mutually exclusive",
		)
	case lease != nil:
		return spooler.Lease(*lease), nil
	}

	name := *leaseFile
	f := os.Stdin
	if name != "-" {
		var err error
		f, err = os.Open(name)
		if err != nil {
			return "", err
		}
		defer f.Close()
	}
	var record struct {
		Lease string `json:"lease"`
	}
	if err := json.NewDecoder(f).Decode(&record); err != nil {
		return "", fmt.Errorf(
			"decode %s: %w",
			name, err,
		)
	}
	if record.Lease == "" {
		return "", cli.Exitf(2,
			"no lease in %s",
			name,
		)
	}
	return spooler.Lease(record.Lease), nil
}

// _leaseOpCommand is the shape of the verbs that present a lease: ack, nack,
// release and fail. It owns their flags and resolves the lease; a verb embeds
// it and adds its SDK call through run.
type _leaseOpCommand struct {
	config *config

	spool     string
	lease     *string
	leaseFile *string
}

func (c *_leaseOpCommand) DefineFlags(fs *flag.FlagSet) {
	defineSpoolFlag(&c.spool, fs)
	defineLeaseFlag(&c.lease, fs)
	defineLeaseFileFlag(&c.leaseFile, fs)
}

// run resolves the lease from the flags and hands it with the spool to fn,
// the verb's SDK call.
func (c *_leaseOpCommand) run(ctx context.Context, args []string, fn func(spool string, lease spooler.Lease) error) error {
	if err := ensureNoArgs(args); err != nil {
		return err
	}
	if noFlags(ctx) {
		return new(cli.UsageError)
	}
	lease, err := readLease(c.lease, c.leaseFile)
	if err != nil {
		return err
	}
	return fn(c.spool, lease)
}
