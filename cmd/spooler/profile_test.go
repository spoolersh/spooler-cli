package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spoolersh/spooler-go"
)

// TestProfile covers the root's own flags: the profile file, the conflict
// between a flag and the profile, and the environment twins.
func TestProfile(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(config, []byte(""+
		"profiles:\n"+
		"  canary:\n"+
		"    host: canary.example:8443\n"+
		"    api-key: k1\n"+
		"    spool: main\n",
	), 0600); err != nil {
		t.Fatal(err)
	}
	described := spoolOutput{
		Name: "main",
	}
	for _, test := range []struct {
		name string
		args []string
		env  map[string]string
		// expSpool is the spool the fake must see; empty means no call.
		expSpool string
		// expHost is the host the client is built with; empty is unchecked.
		expHost string
		// expNoTLS is whether the client is built without TLS.
		expNoTLS bool
		exp      result
	}{
		{
			name:     "profile fills the spool",
			args:     []string{"--config", config, "--profile", "canary", "stats"},
			expSpool: "main",
			exp: result{
				stdout: record(t, described),
			},
		},
		{
			name:     "flag over profile",
			args:     []string{"--config", config, "--profile", "canary", "stats", "-s", "other"},
			expSpool: "other",
			exp: result{
				stdout: record(t, described),
			},
		},
		{
			// Not a bare invocation: the profile gave the spool, so the
			// reply names the flag still missing, not the usage.
			name: "profile fills the spool but not the payload",
			args: []string{"--config", config, "--profile", "canary", "send"},
			exp: result{
				stderr: "no payload: pass --data, --data-file, or `--data-file -` for stdin\n",
				code:   2,
			},
		},
		{
			name: "unknown profile",
			args: []string{"--config", config, "--profile", "nope", "stats"},
			exp: result{
				stderr: "--profile: select profile \"nope\": no profile defined in " + config + "\n",
				code:   2,
			},
		},
		{
			name:     "flag over profile host",
			args:     []string{"--config", config, "--profile", "canary", "--host", "other:1", "stats"},
			expSpool: "main",
			expHost:  "other:1",
			exp: result{
				stdout: record(t, described),
			},
		},
		{
			name: "profile over environment",
			args: []string{"--config", config, "--profile", "canary", "stats"},
			env: map[string]string{
				"SPOOLER_HOST":  "env.example:1",
				"SPOOLER_SPOOL": "other",
			},
			expSpool: "main",
			expHost:  "canary.example:8443",
			exp: result{
				stdout: record(t, described),
			},
		},
		{
			name: "environment fills what the profile lacks",
			args: []string{"--config", config, "--profile", "canary", "stats"},
			env: map[string]string{
				"SPOOLER_NO_TLS": "true",
			},
			expSpool: "main",
			expHost:  "canary.example:8443",
			expNoTLS: true,
			exp: result{
				stdout: record(t, described),
			},
		},
		{
			name: "missing config file",
			args: []string{"--config", filepath.Join(dir, "none.yaml"), "--profile", "canary", "stats"},
			exp: result{
				stderr: "--config: select profile \"canary\": open " + filepath.Join(dir, "none.yaml") + ": no such file or directory\n",
				code:   2,
			},
		},
		{
			name: "environment twins",
			args: []string{"stats"},
			env: map[string]string{
				"SPOOLER_CONFIG":  config,
				"SPOOLER_PROFILE": "canary",
			},
			expSpool: "main",
			exp: result{
				stdout: record(t, described),
			},
		},
		{
			name: "spool from the environment",
			args: []string{"stats"},
			env: map[string]string{
				"SPOOLER_SPOOL": "main",
			},
			expSpool: "main",
			exp: result{
				stdout: record(t, described),
			},
		},
		{
			// Only the context flags have a twin: an output choice, which
			// would be a usage error as a flag here, a lease or a consent
			// left in the environment reach no command.
			name: "only the context has a twin",
			args: []string{"stats"},
			env: map[string]string{
				"SPOOLER_SPOOL":   "main",
				"SPOOLER_OUTPUT":  "table",
				"SPOOLER_HUMAN":   "true",
				"SPOOLER_LEASE":   "L9",
				"SPOOLER_YES":     "true",
				"SPOOLER_VERSION": "true",
			},
			expSpool: "main",
			exp: result{
				stdout: record(t, described),
			},
		},
		{
			name:     "log level",
			args:     []string{"--log-level", "debug", "stats", "-s", "main"},
			expSpool: "main",
			exp: result{
				stdout: record(t, described),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var spool string
			fake := &fakeClient{
				DoSpool: func(_ context.Context, r spooler.SpoolRequest) (spooler.Spool, error) {
					spool = r.Spool
					return spooler.Spool{
						Name: "main",
					}, nil
				},
			}
			act := runEnv(t, fake, test.env, "", test.args...)
			expect(t, act, test.exp)
			if spool != test.expSpool {
				t.Errorf("spool: %q; want %q", spool, test.expSpool)
			}
			if test.expHost != "" && fake.sdk.Host != test.expHost {
				t.Errorf("host: %q; want %q", fake.sdk.Host, test.expHost)
			}
			if fake.sdk != nil && fake.sdk.InsecureDisableTLS != test.expNoTLS {
				t.Errorf("no tls: %v; want %v",
					fake.sdk.InsecureDisableTLS, test.expNoTLS,
				)
			}
		})
	}
}

// TestDebug pins that --debug alone points the SDK's dump at stderr: not
// the log level, and not the environment, since the flag has no twin.
func TestDebug(t *testing.T) {
	for _, test := range []struct {
		name     string
		args     []string
		env      map[string]string
		expDebug bool
	}{
		{
			name: "omitted",
			args: []string{"stats", "-s", "main"},
		},
		{
			name:     "flag",
			args:     []string{"--debug", "stats", "-s", "main"},
			expDebug: true,
		},
		{
			name: "no environment twin",
			args: []string{"stats", "-s", "main"},
			env: map[string]string{
				"SPOOLER_DEBUG": "true",
			},
		},
		{
			name: "log level alone dumps nothing",
			args: []string{"--log-level", "debug", "stats", "-s", "main"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeClient{
				DoSpool: func(context.Context, spooler.SpoolRequest) (spooler.Spool, error) {
					return spooler.Spool{
						Name: "main",
					}, nil
				},
			}
			act := runEnv(t, fake, test.env, "", test.args...)
			if act.code != 0 {
				t.Fatalf("exit: %d; want 0\nstderr:\n%s", act.code, act.stderr)
			}
			if debug := fake.sdk.DebugWriter == os.Stderr; debug != test.expDebug {
				t.Errorf("debug: %v; want %v", debug, test.expDebug)
			}
		})
	}
}
