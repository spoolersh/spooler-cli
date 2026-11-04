package main

import (
	"io"
	"os"
	"strings"

	"github.com/gobwas/cli"
	"github.com/spoolersh/spooler-go"
)

// readData resolves the payload the send verbs take: the --data value, or
// the --data-file file, "-" for stdin. Neither or both is a usage error
// naming the three forms. A nil flag is one omitted; an empty value is a
// value, an empty message or a path the OS refuses. The closer, when not
// nil, is the file opened and is closed once the request is done.
func readData(data, dataFile *string) (spooler.DataSource, io.Closer, error) {
	switch {
	case data != nil && dataFile != nil:
		return spooler.DataSource{}, nil, cli.Exitf(2,
			"--data and --data-file are mutually exclusive",
		)

	case data == nil && dataFile == nil:
		return spooler.DataSource{}, nil, cli.Exitf(2,
			"no payload: pass --data, --data-file, or `--data-file -` for stdin",
		)

	case data != nil:
		return spooler.DataSourceReadSeeker(strings.NewReader(*data)), nil, nil

	case *dataFile == "-":
		// NOTE: stdin is not seekable, so no retries are possible.
		return spooler.DataSourceReader(os.Stdin), nil, nil
	}

	file, err := os.Open(*dataFile)
	if err != nil {
		return spooler.DataSource{}, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return spooler.DataSource{}, nil, err
	}
	if info.Mode().IsRegular() {
		// A regular file can be rewound, which enables retries.
		return spooler.DataSourceReadSeeker(file), file, nil
	}
	// A pipe, FIFO or device (<(cmd), /dev/stdin) can't seek: the SDK would
	// fail measuring it, so stream it once.
	return spooler.DataSourceReader(file), file, nil
}
