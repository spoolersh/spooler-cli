package main

import (
	"encoding/json"
	"os"

	"github.com/spoolersh/spooler-cli/internal/format"
)

// writeMeta writes the record meta as JSON to dest, the file a --*-file flag
// named; nil is a flag not given and writes nothing. With asFrame the payload
// follows on dest, so an empty line ends the record.
func writeMeta[T any](dest *os.File, meta T, asFrame bool) error {
	if dest == nil {
		return nil
	}
	err := json.NewEncoder(dest).Encode(meta)
	if err == nil && asFrame {
		_, err = dest.WriteString("\n")
	}
	return err
}

// metaEncoder writes the record --include-meta prints on stdout ahead of
// the payload, in the --output format.
type metaEncoder[T any] struct {
	w   *fileTailWriter
	enc format.Encoder[T]
}

// newMetaEncoder builds the encoder out's flags select, writing to f. A
// usage error among the output flags surfaces here, so it runs before the
// operation: a wrong flag must not cost a lease or a request.
func newMetaEncoder[T any](f *os.File, out *output[T]) (*metaEncoder[T], error) {
	w := &fileTailWriter{
		file: f,
	}
	enc, err := out.encoderTo(w)
	if err != nil {
		return nil, err
	}
	return &metaEncoder[T]{
		w:   w,
		enc: enc,
	}, nil
}

// encode writes meta. With asFrame the payload follows, so the record ends
// with the empty line a reader takes as its end; a text record ends so
// already.
func (e *metaEncoder[T]) encode(meta T, asFrame bool) error {
	err := e.enc.Encode(meta)
	if err == nil {
		err = e.enc.Flush()
	}
	if err != nil {
		return err
	}
	if asFrame && !e.w.endsWithEmptyLine() {
		_, err = e.w.Write([]byte{'\n'})
	}
	return err
}
