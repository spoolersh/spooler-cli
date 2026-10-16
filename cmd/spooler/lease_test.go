package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLease(t *testing.T) {
	for _, test := range []struct {
		name  string
		lease string
		file  string
		exp   string
		// errMsg is the exact error wanted, with the file's path spelled
		// FILE; empty with err false means success.
		errMsg string
		err    bool
	}{
		{
			name:  "lease value",
			lease: "L1",
			exp:   "L1",
		},
		{
			name: "lease file",
			file: `{"id":"m1","retries":2,"lease":"L1","dataSize":42}` + "\n",
			exp:  "L1",
		},
		{
			name: "lease file followed by the payload",
			file: `{"id":"m1","lease":"L1"}` + "\n\nPAYLOAD",
			exp:  "L1",
		},
		{
			name: "other members are ignored",
			file: `{"lease":"L1","leaseExpiresAt":"not a time"}`,
			exp:  "L1",
		},
		{
			name:   "no lease member",
			file:   `{"id":"m1"}`,
			errMsg: "no lease in FILE",
		},
		{
			name: "not json",
			file: "PAYLOAD",
			err:  true,
		},
		{
			name:   "neither flag",
			errMsg: "no lease: pass --lease or --lease-file",
		},
		{
			name:   "both flags",
			lease:  "L1",
			file:   `{"lease":"L1"}`,
			errMsg: "--lease and --lease-file are mutually exclusive",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var name string
			if test.file != "" {
				name = filepath.Join(t.TempDir(), "lease.json")
				if err := os.WriteFile(name, []byte(test.file), 0600); err != nil {
					t.Fatalf("WriteFile: %v", err)
				}
			}
			var lease, file *string
			if test.lease != "" {
				lease = &test.lease
			}
			if name != "" {
				file = &name
			}
			act, err := readLease(lease, file)
			switch {
			case test.errMsg != "":
				exp := strings.ReplaceAll(test.errMsg, "FILE", name)
				if err == nil || err.Error() != exp {
					t.Fatalf("error: %v; want %q", err, exp)
				}
				return
			case test.err:
				if err == nil {
					t.Fatal("error: nil; want one")
				}
				return
			case err != nil:
				t.Fatalf("error: %v; want nil", err)
			}
			if string(act) != test.exp {
				t.Errorf("lease: %q; want %q", act, test.exp)
			}
		})
	}
	t.Run("missing file", func(t *testing.T) {
		name := filepath.Join(t.TempDir(), "nope.json")
		_, err := readLease(nil, &name)
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("error: %v; want not exist", err)
		}
	})
}
