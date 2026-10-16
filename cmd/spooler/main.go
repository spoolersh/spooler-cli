package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"text/template"

	"github.com/gobwas/cli"
	"github.com/gobwas/flagutil"
	"github.com/gobwas/flagutil/parse/env"
	"github.com/gobwas/flagutil/parse/pargs"
	"github.com/spoolersh/spooler-cli/internal/flagvars"
	"github.com/spoolersh/spooler-go"
)

type config struct {
	client client
}

// The root's help has a prolog before the usage and an epilog after the
// flags, templates like a command's description. The prolog opens with the
// wordmark as block characters, 80 columns by 6 rows: a copy of the brand
// package's logo.txt, which is rendered there from the wordmark SVG.

//go:embed main_prolog.txt
var mainPrologText string

//go:embed main_epilog.txt
var mainEpilogText string

var (
	mainProlog *template.Template
	mainEpilog *template.Template
)

func init() {
	mainProlog = must(template.New("").Parse(mainPrologText))
	mainEpilog = must(template.New("").Parse(mainEpilogText))
}

func main() {
	// Unasked for, SIGPIPE from a write to a closed stdout ends the process at
	// once, before recv can release the lease it holds; asked for, the write
	// fails with EPIPE instead and the error takes the release path.
	signal.Notify(make(chan os.Signal, 1), syscall.SIGPIPE)

	r, cmd := app(func(c *spooler.Client) client {
		return c
	})
	r.Main(cmd)
}

// app builds the command tree and its runner. newClient turns the SDK client
// the flags configured into what the commands call, so that tests put a stub
// there.
func app(newClient func(*spooler.Client) client) (*cli.Runner, cli.Command) {
	var (
		// No logs by default -- slog.ErrorLevel is 8.
		logLevel slog.Level = 16
		debug    bool

		showVersion bool

		host   *string
		apiKey *string
		noTLS  *bool

		profile profileEntry

		config config
	)
	queues := mustGroup(commandGroup{
		summary: "Manage the spool's queues",
		cmds: cli.Commands{
			"list": &queuesListCommand{
				config: &config,
				output: makeOutput("table", queueColumns),
			},
			"describe": &queuesDescribeCommand{
				config: &config,
				output: makeOutput[queueOutput]("json", nil, "text", "json"),
			},
			"delete": &queuesDeleteCommand{
				config: &config,
			},
			"create": &queuesCreateCommand{
				config: &config,
				output: makeOutput[queueOutput]("json", nil, "text", "json"),
			},
			"update": &queuesUpdateCommand{
				config: &config,
			},
		},
		order: []string{"list", "describe", "create", "update", "delete"},
		aliases: map[string]string{
			"ls": "list",
		},
	})
	cmds := cli.Commands{
		"queues": queues,
		"send": &sendCommand{
			config: &config,
			output: makeOutput[sendResultOutput]("json", nil, "text", "json"),
		},
		"recv": &recvCommand{
			config: &config,
			output: makeOutput[recvResultOutput]("json", nil, "text", "json"),
		},
		"ack": &ackCommand{
			config: &config,
		},
		"nack": &nackCommand{
			config: &config,
		},
		"release": &releaseCommand{
			config: &config,
		},
		"fail": &failCommand{
			config: &config,
		},
		"renew": &renewCommand{
			config: &config,
			output: makeOutput[renewResultOutput]("json", nil, "text", "json"),
		},
		"inspect": &inspectCommand{
			config: &config,
			output: makeOutput[inspectResultOutput]("json", nil, "text", "json"),
		},
		"stats": &statsCommand{
			config: &config,
			output: makeOutput[spoolOutput]("json", nil, "text", "json"),
		},
		"discard": &discardCommand{
			config: &config,
		},
		"recover": &recoverCommand{
			config: &config,
		},
		"messages": &messagesCommand{
			config: &config,
			output: makeOutput("table", messageColumns),
		},
		"version": &versionCommand{
			output: makeOutput[versionOutput]("json", nil, "text", "json"),
		},
	}
	root := mustGroup(commandGroup{
		prolog: renderDesc(mainProlog),
		cmds:   cmds,
		order: []string{
			"send",
			"recv",
			"ack",
			"nack",
			"release",
			"fail",
			"renew",
			"inspect",
			"recover",
			"discard",
			"messages",
			"queues",
			"stats",
			"version",
		},
		aliases: map[string]string{
			"receive": "recv",
		},
		epilog: renderDesc(mainEpilog),
	})
	cmd := cli.Container{
		// Command gives the root help its synopsis and the command list, in
		// the order of the work; DoRun still wraps the dispatch with the
		// connection setup.
		Command: root,
		DoDefineFlags: func(fs *flag.FlagSet) {
			defineLevelFlag(&logLevel, fs)
			fs.BoolVar(&debug,
				"debug", false,
				"dump every request and response, payloads included, on"+
					" stderr in the SDK's form; the API key, leases and"+
					" handles are redacted (debugging only)",
			)

			fs.Var(flagvars.OptVar(&host, flagvars.Ident),
				"host",
				"api host as host[:port]",
			)
			fs.Var(flagvars.OptVar(&apiKey, flagvars.Ident),
				"api-key",
				"API key; visible in ps and shell history, where the"+
					" environment variable is not",
			)
			fs.Var(flagvars.OptVar(&noTLS, strconv.ParseBool),
				"no-tls",
				"do not use tls for communication with spooler server"+
					" (testing only)",
			)
			fs.BoolVar(&showVersion,
				"version", false,
				"print the CLI's version and the SDK's, as the version command"+
					" does",
			)

			// Below are two flags used by profile parser.
			// Only the parser looks them up and uses values.
			{
				fs.String(
					"config", defaultConfigPath(),
					"path to a profiles configuration file",
				)
				fs.String(
					"profile", "",
					"name of the profile to use from the configuration file",
				)
			}
		},
		DoRun: func(ctx context.Context, args []string) error {
			if showVersion {
				return mapExit(ctx, root.Run(ctx, []string{"version"}))
			}
			logger := makeLogger(logLevel)

			if host == nil {
				host = new(spooler.DefaultHost)
			}
			c := &spooler.Client{
				Trace: spooler.LoggerClientTrace(logger),
				Host:  *host,
			}
			if apiKey != nil {
				c.APIKey = *apiKey
			}
			if noTLS != nil && *noTLS {
				c.InsecureDisableTLS = true
			}
			if debug {
				c.DebugWriter = os.Stderr
			}
			config.client = newClient(c)

			ctx = with(ctx, &config)
			ctx = with(ctx, logger)

			return mapExit(ctx, root.Run(ctx, args))
		},
	}
	r := cli.Runner{
		TermSignals: []os.Signal{
			syscall.SIGINT,
			syscall.SIGTERM,
		},
		DoParseFlags: func(ctx context.Context, fs *flag.FlagSet, args []string) ([]string, error) {
			opts, rest := flagParseOptions(fs, args, &profile)
			if err := flagutil.Parse(ctx, fs, opts...); err != nil {
				return nil, parseError(err)
			}
			return rest(), nil
		},
		DoPrintFlags: func(ctx context.Context, w io.Writer, fs *flag.FlagSet) error {
			orig := fs.Output()
			fs.SetOutput(w)
			defer fs.SetOutput(orig) // Restore original output writer.
			opts, _ := flagParseOptions(fs, nil, &profile)
			return flagutil.PrintDefaults(ctx, fs, opts...)
		},
	}
	return &r, &cmd
}

// parseError puts a flag parse error in the CLI's own voice: no library
// prefix, and the flag with its dashes, as every message names one: the name
// the parser read, a shorthand included, since that is the one the caller
// typed, where the flag's own name may be one this parser does not take. Any
// other error, help asked for by a flag among them, passes through.
func parseError(err error) error {
	e, ok := errors.AsType[*flagutil.ParseError](err)
	if !ok {
		return err
	}
	if e.Undefined {
		return fmt.Errorf(
			"unknown flag: %s",
			dashed(e.Name),
		)
	}
	return fmt.Errorf(
		"%s: %w",
		dashed(e.Name), e.Err,
	)
}

// dashed is a flag name as it is typed: one dash for a shorthand, two
// otherwise.
func dashed(name string) string {
	if len(name) == 1 {
		return "-" + name
	}
	return "--" + name
}

func makeLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "error" {
				if a.Value.Any() == nil {
					// Drop zero error attrs.
					return slog.Attr{}
				}
			}
			return a
		},
	}))
}

func flagParseOptions(fs *flag.FlagSet, args []string, profile *profileEntry) (
	opts []flagutil.ParseOption,
	rest func() []string,
) {
	p := &pargs.Parser{
		Args: args,
	}
	opts = []flagutil.ParseOption{
		flagutil.WithParser(p,
			// The shorthand, where a flag has one, ahead of the name, so
			// help prints "-s, --spool". Lowercase names the value itself,
			// uppercase a file that holds it, read or written.
			flagutil.WithNameFunc(func(name string) []string {
				switch name {
				case "columns":
					return []string{"c", name}
				case "config":
					return []string{"C", name}
				case "data":
					return []string{"d", name}
				case "data-file":
					return []string{"D", name}
				case "file":
					return []string{"f", name}
				case "include-meta":
					return []string{"i", name}
				case "lease":
					return []string{"l", name}
				case "lease-file":
					return []string{"L", name}
				case "limit":
					return []string{"n", name}
				case "match":
					return []string{"m", name}
				case "output":
					return []string{"o", name}
				case "profile":
					return []string{"p", name}
				case "queue":
					return []string{"q", name}
				case "spool":
					return []string{"s", name}
				}
				return []string{name}
			}),
		),
		// The environment twice around the profile: first the profile's
		// own selection, then, once the profile has set what the command
		// line left unset, the rest, so a flag beats the profile and the
		// profile beats a variable. Only the context has a variable: where,
		// as whom and how loudly the CLI runs, and the spool and queue a
		// session works in, as git and kubectl read their environment.
		// Everything else shapes one invocation and is asked for on its
		// command line.
		flagutil.WithParser(
			&env.Parser{
				Prefix:       "SPOOLER_",
				SetSeparator: "_",
			},
			flagutil.WithNameFunc(func(name string) []string {
				switch name {
				case
					"config",
					"profile":
					return []string{name}
				}
				return nil
			}),
		),
		flagutil.WithParser(&profileParser{
			dest: profile,
		}),
		flagutil.WithParser(
			&env.Parser{
				Prefix:       "SPOOLER_",
				SetSeparator: "_",
			},
			flagutil.WithNameFunc(func(name string) []string {
				switch name {
				case
					"api-key",
					"host",
					"log-level",
					"no-tls",
					"queue",
					"spool":
					return []string{name}
				}
				return nil
			}),
		),
	}
	opts = append(opts,
		flagutil.WithCustomUsage(),
		flagutil.WithUsageOrder(compareFlags),
	)
	return opts, p.NonOptionArgs
}

func configDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "spooler"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "spooler"), nil
}

func defaultConfigPath() string {
	dir, err := configDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "config.yaml")
}
