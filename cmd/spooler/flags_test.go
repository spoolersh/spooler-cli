package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spoolersh/spooler-go"
)

func TestEmptyValue(t *testing.T) {
	// An empty value is a value the caller passed, not an omitted flag: an
	// empty one beside its pair is still the pair, a lease goes to the SDK
	// as it is, which refuses the empty one, and a path to the OS, which
	// refuses it before any request.
	leaseFile := filepath.Join(t.TempDir(), "lease.json")
	if err := os.WriteFile(leaseFile, []byte(`{"lease":"L9"}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Run("empty lease beside a lease file", func(t *testing.T) {
		act := run(t, &fakeClient{}, "", "ack", "-s", "main", "--lease", "", "--lease-file", leaseFile)
		expect(t, act, result{
			stderr: "--lease and --lease-file are mutually exclusive\n",
			code:   2,
		})
	})
	t.Run("empty lease file beside a lease", func(t *testing.T) {
		act := run(t, &fakeClient{}, "", "ack", "-s", "main", "--lease", "L9", "--lease-file", "")
		expect(t, act, result{
			stderr: "--lease and --lease-file are mutually exclusive\n",
			code:   2,
		})
	})
	t.Run("lease goes to the sdk as given", func(t *testing.T) {
		lease := "unset"
		fake := &fakeClient{
			DoAck: func(_ context.Context, r spooler.AckRequest) error {
				lease = string(r.Lease)
				return nil
			},
		}
		act := run(t, fake, "", "ack", "-s", "main", "--lease", "")
		expect(t, act, result{})
		if lease != "" {
			t.Errorf("lease: %q; want %q", lease, "")
		}
	})
	t.Run("empty handle beside a lease file", func(t *testing.T) {
		act := run(t, &fakeClient{}, "", "discard", "-s", "main", "--handle", "", "--lease-file", leaseFile, "--yes")
		expect(t, act, result{
			stderr: "--handle and --lease or --lease-file are mutually exclusive\n",
			code:   2,
		})
	})
	t.Run("handle goes to the sdk as given", func(t *testing.T) {
		handle := "unset"
		fake := &fakeClient{
			DoRecover: func(_ context.Context, r spooler.RecoverRequest) error {
				handle = string(r.Handle)
				return nil
			},
		}
		act := run(t, fake, "", "recover", "-s", "main", "--handle", "")
		expect(t, act, result{})
		if handle != "" {
			t.Errorf("handle: %q; want %q", handle, "")
		}
	})
	t.Run("lease file goes to the os before the receive", func(t *testing.T) {
		act := run(t, &fakeClient{}, "", "recv", "-s", "main", "-q", "jobs", "--lease-file", "")
		expect(t, act, result{
			stderr: "open : no such file or directory\n",
			code:   4,
		})
	})
	t.Run("data file goes to the os before the request", func(t *testing.T) {
		act := run(t, &fakeClient{}, "", "inspect", "-s", "main", "--handle", "H1", "-D", "")
		expect(t, act, result{
			stderr: "open : no such file or directory\n",
			code:   4,
		})
	})
}
