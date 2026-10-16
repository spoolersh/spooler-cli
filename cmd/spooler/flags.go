package main

import (
	"flag"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/spoolersh/spooler-cli/internal/flagvars"
	"github.com/spoolersh/spooler-go"
)

// The spool and the queue are Values rather than StringVars, so the help
// prints no type line for them: "string" says nothing a person needs.
func defineSpoolFlag(p *string, fs *flag.FlagSet) {
	fs.Var(flagvars.StringVar(p),
		"spool",
		"spool to work with",
	)
}
func defineQueueFlag(p *string, fs *flag.FlagSet) {
	fs.Var(flagvars.StringVar(p),
		"queue",
		"queue to work with",
	)
}

// The identity flags track presence: an empty value is a value, which the
// API judges, not an omitted flag its pair may stand in for.
func defineHandleFlag(p **string, fs *flag.FlagSet) {
	fs.Var(flagvars.OptVar(p, flagvars.Ident),
		"handle",
		"a message handle",
	)
}
func defineLeaseFlag(p **string, fs *flag.FlagSet) {
	fs.Var(flagvars.OptVar(p, flagvars.Ident),
		"lease",
		"the message's lease, as recv returned it; visible in ps and shell"+
			" history, where --lease-file is not",
	)
}

func defineMessageStateFlag(p *spooler.MessageState, d spooler.MessageState, fs *flag.FlagSet) {
	*p = d
	fs.Var(flagvars.ConvFuncVar(p, spooler.ParseMessageState),
		"state",
		"keep only messages in this `state`: delayed, visible, pending, failed"+
			" or unavailable",
	)
}

func defineQueueDeleteModeFlag(p *spooler.QueueDeleteMode, fs *flag.FlagSet) {
	fs.Var(flagvars.ConvFuncVar(p, spooler.ParseQueueDeleteMode),
		"mode",
		"how to delete, as a `mode`: force deletes the queue and every message"+
			" in it at once; drain refuses new messages and deletes the queue"+
			" once consumers have emptied it",
	)
}

// valueOr returns the value a tracked flag was given, or d when it was not.
func valueOr(p *string, d string) string {
	if p == nil {
		return d
	}
	return *p
}

func defineFileFlag(p **string, fs *flag.FlagSet) {
	fs.Var(flagvars.OptVar(p, flagvars.Ident),
		"file",
		"file holding a queue record, as queues describe -o json prints it;"+
			" its name and settings are read, and a flag given overrides a"+
			" value in it",
	)
}

func defineLimitFlag(p *int, fs *flag.FlagSet) {
	fs.Var(flagvars.ConvFuncVar(p, parseCount),
		"limit",
		"print at most this `count` of items, fetching no more than needed;"+
			" 0 means every item",
	)
}

// The uid flag tracks presence and maps onto the SDK's pointer as given; an
// empty one goes to the API, which refuses it.
func defineUIDFlag(p **string, fs *flag.FlagSet) {
	fs.Var(flagvars.OptVar(p, flagvars.Ident),
		"uid",
		"act only on the queue with this uid, as queues describe reports it;"+
			" another queue under the name is refused and nothing changes",
	)
}

func defineYesFlag(p *bool, fs *flag.FlagSet) {
	fs.BoolVar(p,
		"yes", false,
		"confirm a destructive operation; without it nothing is changed,"+
			" and nothing ever prompts",
	)
}

func defineReturnHandleFlag(p *bool, d bool, fs *flag.FlagSet) {
	fs.BoolVar(p,
		"return-handle", d,
		"return a handle to each message; a handle is a credential, so off"+
			" by default",
	)
}

func defineDelayFlag(p **int, fs *flag.FlagSet) {
	fs.Var(flagvars.OptVar(p, parseWholeSeconds),
		"delay",
		"`duration` until the message becomes visible (30s, 5m); omitted"+
			" means at once",
	)
}

func defineDataFileFlag(p **string, fs *flag.FlagSet) {
	fs.Var(flagvars.OptVar(p, flagvars.Ident),
		"data-file",
		"file to write the payload to instead of stdout",
	)
}
func defineLeaseFileFlag(p **string, fs *flag.FlagSet) {
	fs.Var(flagvars.OptVar(p, flagvars.Ident),
		"lease-file",
		"file holding the lease, as recv --lease-file wrote it",
	)
}

func defineIncludeMetaFlag(p *bool, fs *flag.FlagSet) {
	fs.BoolVar(p,
		"include-meta", false,
		"print the message's attributes on stdout ahead of the payload, in the"+
			" --output format",
	)
}

func defineOverwriteFlag(p *bool, fs *flag.FlagSet) {
	fs.BoolVar(p,
		"overwrite", false,
		"replace an existing file named by a --*-file flag",
	)
}

func defineLevelFlag(p *slog.Level, fs *flag.FlagSet) {
	fs.Var((*levelFlag)(p),
		"log-level",
		"lowest `level` of logs to print on stderr: debug, info, warn or"+
			" error; off prints none",
	)
}

// levelFlag is --log-level: a slog level, or off, above every level.
type levelFlag slog.Level

// levelOff is above every slog level, so nothing is logged.
const levelOff = levelFlag(slog.LevelError + 8)

func (l *levelFlag) String() string {
	if *l >= levelOff {
		return "off"
	}
	return strings.ToLower(slog.Level(*l).String())
}

func (l *levelFlag) Set(s string) error {
	switch strings.ToLower(s) {
	case "debug":
		*l = levelFlag(slog.LevelDebug)
	case "info":
		*l = levelFlag(slog.LevelInfo)
	case "warn":
		*l = levelFlag(slog.LevelWarn)
	case "error":
		*l = levelFlag(slog.LevelError)
	case "off":
		*l = levelOff
	default:
		return fmt.Errorf(
			"unexpected log level: %q",
			s,
		)
	}
	return nil
}

func defineMatchFlag(p **regexp.Regexp, fs *flag.FlagSet) {
	fs.Var(
		flagvars.ConvFuncVar(p, regexp.Compile),
		"match",
		"a regular expression to match results",
	)
}
