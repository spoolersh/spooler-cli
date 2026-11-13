# Agent instructions

## What this is

The `spooler` command: a CLI for Spooler, a hosted message queue. A thin
wrapper over the Go client library, `github.com/spoolersh/spooler-go`: every
command is the library's calls plus input parsing and output rendering, nothing
else.

It serves humans and AI agents equally: the same commands, output and exit
codes; nothing switches format behind the caller's back.

Priorities: correctness > predictability > ergonomics > features. The CLI adds
no guarantee and hides none: it never acks, renews or settles unless a flag
asks, and adds no retry of its own; the library's retries stay at their
default, covering only the conditions the docs class as transient and as having
changed nothing.

## North stars: git and curl

When a design question is open and this file is silent, do what git does; for
connection and compatibility, what curl does. Where they and this file differ,
this file wins.

- **git:** options that say where and as whom it runs go before the command
  (`git -C dir`); everything else belongs to the command that reads it and
  follows its name (`git log --oneline`). Here: `spooler --host H --api-key K
  queues list --spool S -o json`.
- **git:** machine output is a separate, stable contract (`--porcelain`); here
  it is `-o json`, and the `text` format may improve freely.
- **git:** `git help <command>` and `<command> -h` give the same per-command
  help. It never prompts where a flag can say yes (`git clean -f`); here,
  `--yes`.
- **go:** the reply to an unknown command is two lines and no guess: `go
  buidl: unknown command` then `Run 'go help' for usage.`. With a command
  list that fits one screen, the help is the suggestion.
- **curl:** a released flag, output or exit code keeps working forever; every
  exit code is documented; data goes to stdout, diagnostics only when asked.

## Where truth lives

- **The public docs at https://docs.spooler.sh are the contract** for what the
  API does: operations, kinds, statuses, limits, retry rules. This file never
  restates API behaviour; when a task needs a fact about the API, read the docs
  page and cite it. When docs and the evaluation server disagree or the docs
  are silent, ask the project owner.
- **The Go library is the only client.** No HTTP from this repo: what the
  library lacks is a library change first. The library hides the transport and
  so does the CLI: no URLs, status codes or other HTTP notions in its own
  flags, output or words. A library message passes through whatever it names
  (`Post "https://...": connection refused`), as curl names the host it could
  not reach.
- **This CLI's own contract** -- command names, flags, output shapes, exit
  codes -- is this file plus the README, and is public from v0.1: a change to
  any of them is a breaking change.
- **Rules, decisions and conventions live, in this order, in the source
  code, its doc comments, the README and this file.** What an earlier
  source states, a later one does not repeat; this file holds what the
  code cannot say: decisions, the why behind them, and what is forbidden.

## This repo is public

Everything here is a client of the documented API and nothing else. Never let
in: internal hostnames, topology, replication or storage details, limits typed
from memory, pricing, roadmap, or anything from the private repos' source.
Every statement must be checkable with an API key.

The in-memory image `spoolersh/memspoolerd` is NOT open source: it ships
under the memspoolerd License (run it for your own purposes, production
included; never provide it to others or host it as a service). Never call it
"OSS" or "open source"; say "free to run yourself". The CLI itself is open
source.

## Vocabulary

Every command, flag and output field uses the docs' words; the CLI coins none.
Grep the docs before naming anything.

- Verbs mirror the API's operations: `send`, `recv` (alias `receive`), `ack`,
  `nack`, `release`, `fail`, `renew`, `inspect`, `recover`, `discard`. `list`
  (alias `ls`) and `describe` for reads.
- "Failed" is a message state, not a noun: failed messages are `messages
  --queue q --state failed`.
- Sign in/up/out, never login/logout. `spooler login` is not a command, not
  even an alias: it is unknown, and the reply points at the help.
- Flags mirror the request parameters they set, named without the unit
  suffix: `--delay` for `delaySeconds`, `--lease-timeout` for
  `leaseTimeoutSeconds`, `--uid` (the precondition on update and delete),
  `--wait`. A rate is written in the human form, `100/s`, since
  `intervalMicros` is not one a person writes; `unlimited` removes it, the
  word `--human` prints for none.
- `-o json` field names are the API's JSON names (`id`, `lease`,
  `leaseExpiresAt`, `retries`, `state`, `handle`, ...), so a reader of the docs
  can predict the output without reading the CLI's.

## Commands (v1: data plane only)

`spooler help` and each command's `-h` are the synopsis. The decisions
behind them:

- **Everything is an explicit flag**: `--queue` where the operation names a
  queue, `--lease`, `--handle`. No positional operands for API objects: one
  form in docs, scripts and transcripts.
- **A flag that silently does nothing is refused**: a command that prints
  nothing on success takes no output flag, so `ack -o json` is an unknown flag,
  and an output flag a mode does not fit is a usage error.
- **Payload** is `--data` or `--data-file`, never stdin unasked: a cron job's
  stdin is `/dev/null` and the API accepts an empty body, so an omitted flag
  would enqueue silently. Two flags rather than curl's `--data @file`, since a
  payload may start with `@`.
- **Presence, never format.** A flag whose presence the code acts on is
  tracked by `flagvars.OptVar` or `flagvars.Opt`, never read back through
  its zero value: an empty value is a value the caller passed and goes out
  as given, so `--dedup-string "$ID"` with `ID` unset cannot pass as a send
  without a key, and `--lease "$LEASE" --lease-file F` is refused as the
  pair it is. The CLI never checks a value's format: uid length or
  alphabet, lease shape, hash hex. **The test for whether a check belongs
  in the CLI: could it move into the library without the CLI changing? If
  yes, it is validation on the library's behalf and belongs there. If no,
  because the library cannot observe the condition, it is translation and
  belongs here.**
- **A receive never leases silently**: on a queue without a lease timeout
  an unsettled lease parks the message as pending, so `recv` requires
  `--ack`, `--lease-file` or `--include-meta`, checked before the receive.
- **Shorthands** exist only for flags a person types often. Twins: where a
  flag takes a value and another a file holding it, the value's letter is
  lowercase and the file's uppercase (`-l`/`-L`, `-d`/`-D`). A flag without
  a twin takes its precedent's letter (`-f` kubectl, `-C` git, `-n` `head`).
  `--handle` has none: `-h` is help, and a handle is pasted, not typed. A
  shorthand is a released contract: added only on evidence of use, never
  removed; consent flags (`--yes`, `--ack`, `--overwrite`) never get one,
  their length being part of the consent.
- **Listings are never cut silently**: `--limit 0`, the default, is every
  item. The page size is the server's, never one the CLI asks for, since
  the docs bound `limit` and say nothing beyond the bound. A cursor to
  resume from is not exposed yet.
- **No spool is assumed**, not even one named `default`.
- **Durations**: a bare number is seconds, as `sleep`, `timeout` and systemd
  read one and as the docs' `*Seconds` fields are written; a value that is
  not whole seconds is a usage error, never rounded.
- **Destructive operations** require `--yes`. Never prompt.

Later, in order: `sign-in`/`sign-out` (device flow, account-level access) with
a config file and profiles; account-plane commands (keys, spools). Never:
`tail` (it would lease messages while sounding read-only), more output syntaxes
(`--output yaml`), telemetry, update-available nags, colour themes (every
colour is a palette slot the terminal's theme resolves; no fixed value, no
option to pick one).

## Credentials

The connection mirrors the library's `Client`; `--no-tls` is for the local
evaluation image only. A value is taken from the flag, else the profile, else
the variable. The north stars put the environment above a config file; here a
profile is named on the command line and a variable is ambient, so the named
one wins, and the stale `SPOOLER_API_KEY` beside `--profile prod` loses, where
in the AWS CLI it wins silently. Help never shows a value a profile or a
variable supplies. No hidden default key, not even for localhost, and nothing
is inferred from the host: `localhost` without `--no-tls` still uses TLS.

**The context flags have an environment twin, and no other flag does.** The
context is where, as whom and how loudly the CLI runs, and the spool and queue
a session works in: `--config`, `--profile`, `--host`, `--api-key`, `--no-tls`,
`--log-level`, `--spool` and `--queue`. This is what git, curl and kubectl do,
with no variable for a command's option; the AWS CLI's `AWS_DEFAULT_OUTPUT` is
the exception and the complaint. Everything else shapes one invocation and is
asked for on its command line: consent is never exported; an output choice is
refused by the commands it does not fit, so an exported one would break them;
`--version` exported would replace every command; `--debug` dumps payloads and
is asked for per invocation, as consent is; an exported `--lease-file` would
make `recv` lease on the environment's strength; the rest are preferences a
shell alias holds. `--no-color` has no twin: `NO_COLOR` does the same.

## Output

**stdout carries data; stderr carries errors only.** No progress, hints or
metadata on stderr; diagnostics only when a root flag asks. `--log-level` and
`--debug` are separate flags, since one is readable and the other is anything;
no level turns on the dump. A `--debug-file` waits for a need.

**The destination changes colour and nothing else.** A terminal, a pipe or a
file decides only whether escape sequences are written; bytes, shape,
indentation and fields are the same, so a script reads what a person saw.
Colours are palette slots, never fixed xterm-256 values: only the user's theme
knows the background.

**One output struct, one field walker, four mechanical renderings**: no
templates in the formats, no second syntax, so no matrix. Defaults as gcloud's,
`table` on a listing and `json` on a record, so a script and a person get the
same bytes unflagged. Templates are `--template`, never `--format` and never a
value of `-o`; no `--json` alias.

- **Contracts**: `text`'s is the empty line ending a record, and the lines may
  grow; `json`'s is its fields, added, never renamed or removed, a list being a
  stream of documents and `--compact` being `jq -c`; `table` never changes
  shape for a pipe; `csv` is RFC 4180.
- **`--columns`** takes `-o json` names, so a key copied from `text` output is
  a valid value, as globs that cross dots; a pattern matching no field is a
  usage error listing the fields.
- **`--template`** sees each field as its Go value, nil for one json omits
  (`{{if .handle}}`); a name that is no field is an execution error listing the
  fields, never `<no value>`. No functions beyond `text/template`'s own; any
  added later are permanent.
- **`recv` and `inspect`** write the payload byte for byte, on a terminal too.
  The metadata goes only where a flag sends it, in the format the flag says,
  never the destination, as `curl -i` and `curl -D` do. Where a record and the
  payload share a stream, the record leads and one empty line separates them,
  so a reader takes everything up to the first empty line as the metadata; two
  records on one stream are refused. With `--ack` the ack is sent only after
  the payload is fully written; a failed write, a closed stdout included,
  releases the lease and the error says so.
- **Lease file**: a reader takes any JSON object with a `lease` member and
  ignores the rest; no verb deletes or rewrites it. The `--*-file` flags are
  one family with one mechanism, `fileman.go`, and the prefix names the
  content: `--data-file` the payload, `--lease-file` a record holding a
  credential (mode 0600), `--meta-file` one holding none, `--file` a queue
  record read, never written. No command takes both record flags, so mistaking
  one for the other is an unknown flag, never a lease in the wrong file. An
  existing file is replaced only once the payload is delivered, and kept while
  the lease may still be live.
- **`--human`** is never a default, even at a terminal; it renames nothing
  (`CREATED_AT` keeps its name with an age in the cell) and is refused on the
  contract formats. Which fields have a form is a `human` struct tag, nothing
  inferred from names; the forms are `internal/human`'s, the server's own
  package, whose parsers read every form back.
- **Errors**: human text on stderr, no JSON error object; a library message is
  passed through, never reworded. A usage error with a reason prints its
  sentence alone, since every sentence names the flags to pass; a command given
  no flags prints its help alone; `cli.Usagef` is not used, since a sentence
  followed by the help says everything twice. Each check is its own `if` at the
  top of `Run`, one helper per check, none bundling several; `ensureNoArgs`
  runs before `noFlags`, since a stray word hides every flag behind it from the
  parser and naming it says why.
- **`--help`** orders flags by meaning (`flagrank.go`), not alphabetically:
  `--spool` and `--queue` first, the command's own next, the shared output
  flags last. The help is monochrome, the wordmark in the terminal's own
  colour; the root's help alone carries the wordmark and the exit codes table,
  and a command's `Summary()` appears only in its parent's list.

## Exit codes

The table is in `spooler help` and the mapping in `exit.go`. The numbering: 0,
1 and 2 follow the GNU utilities (grep: 0 match, 1 none, 2 trouble); the
failure classes are spaced, 4 and 8, so a class can gain a neighbour without
renumbering; the unknown result sits apart at 100, the one outcome a caller
must think about rather than retry or give up. None collides with BSD
`sysexits.h` (64-78) or the shell's 126, 127 and 128+N; nothing above 125 is
returned for a result, and an interruption exits with 128 plus the signal's
number, as the shell reports a signalled process.

Exit codes say what to do next; the message on stderr says why. Code 8 comes
after the library's own attempts, so a script backs off for longer than the
library did. Neither the library nor the CLI retries an unknown result; the
calling script decides, on the code. The root sets the code and prints the
message; a command never prints an error itself. `send` and `ack-and-send`
alone add to one, on a result unknown, what the docs' retry rule makes of it:
for `send`, that a retry may send twice without a key the first attempt
carried, or is safe with the same `--dedup-string` or `--dedup-hash` within the
dedup window; for `ack-and-send`, that a retry is safe without a key, since a
retry of a committed call is refused as stale and sends nothing.

TODO: a transport failure that reached no server (a refused connection, a host
that does not resolve) is exit 4 today, since the library keeps the transport
error as text only; it should be 8, which needs the library to mark it with a
condition the CLI can match.

## Libraries and structure

`github.com/gobwas/cli` for commands, `github.com/gobwas/flagutil` for parsing
(its POSIX parser stops at the first non-option argument, so a connection flag
after the command is a usage error) and `github.com/gobwas/glob` for
`--columns`; standard library beyond those, `golang.org/x/*` only if it earns
its place. Rendering lives apart from command logic, so every command renders
through the same writers.

## Coding style

Standard Go, `gofmt`. Comments are concise: what a thing is or does in one
line; a why only when the code cannot say it. One field, one doc comment.
Struct literals one field per line. File modes as the standard library writes
them, `0600`, never `0o600`. Every exported symbol has a doc comment that is a
complete sentence starting with its name. Assign a multi-line call on its own
line, then `if err != nil`. Every `fmt.Errorf` and `cli.Exitf` call has one
shape: the format string on its own line, the arguments on one line, then the
closing parenthesis; a call with no arguments stays on one line when it fits in
80 columns, a tab counting as four; a format string that does not fit is split
with `+`:

```go
return fmt.Errorf(
	"decode %s: %w",
	name, err,
)
return cli.Exitf(2, "no lease: pass --lease or --lease-file")
return cli.Exitf(2,
	"a receive leases the message: pass --ack to settle it,"+
		" --lease-file to keep the lease, or --include-meta to print it",
)
```

`t.Errorf` and `t.Fatalf` follow the same shape unless the whole call fits
in 80 columns, and so does a flag's help string, on its own line after the
name and default:

```go
fs.BoolVar(p,
	"yes", false,
	"confirm a destructive operation; without it nothing is changed,"+
		" and nothing ever prompts",
)
```

A name the caller supplied (a queue, a spool, a profile, a column, a format, a
lease) is `%q`, never `%s`, so an empty or space-bearing value still reads; a
file path is `%s`, as the OS prints it; a list of names is joined and printed
with `%s`. A flag is literal text with its dashes, bare when its dashes delimit
it (`pass --yes`) and in backticks when it has a space inside (`` `-o table`
``, `` `spooler send` ``); an error that asks for a flag names the flag alone,
never a placeholder for its value (`pass --lease or --lease-file`),
placeholders belonging to the synopsis and the help. A `Summary()` is a line in
a list: a capital, an imperative verb, no period ("Send a message to a queue").
A description in `cmd_*.txt` is prose, as a man page's DESCRIPTION: third
person, the first sentence opening with the verb ("Sends a message to a queue
with ...").

## Testing

- Table-driven tests with `testing`; test names are lowercase phrases without
  underscores; case structs hold scenario data only, never mock functions.
  Every case is written one field per line with a `name`, never as a one-line
  literal, so a case reads and diffs like the struct it is.
- Every command is tested end to end in-process: args, env and stdin in;
  stdout, stderr and exit code out, compared exactly. The exit-code table and
  the `-o json` shapes are pinned by these tests.
- Assertions: actual first, then want, never "got": `t.Errorf("exit: %d; want
  %d", act, exp)`. Structs with `cmp.Diff(exp, act)`.
- No integration suite against the evaluation image: the library's own runs the
  wire protocol against it, and what the CLI adds is what the stubbed tables
  pin. The glue in `main`, the real client reaching the commands, is one path,
  checked by hand against a closed port.
- `go test ./...` and `go vet ./...` clean before every hand-off.

## Docs and README

Tone is mq_overview(7): terse, complete, factual. No "simply", "just",
"seamless", "powerful"; never pitch the transport. The README leads with the
local loop (run the evaluation image, export the connection variables, create a
queue, send, recv, ack) and documents the output shapes and exit codes before
the first tag. The "message queues as simple as Go channels" tagline belongs to
the library.

## Releases

Order: server deploy, evaluation image push, library tag, CLI tag: the CLI
never ships behaviour the library and server do not have. Distribution:
`script/install.sh` and `go install`; a Homebrew tap waits for a need.
`SPOOLER_INSTALL_DIR` and `SPOOLER_VERSION` are the script's whole contract, as
released as a flag, and its tests add no variable to it; it never uses `sudo`
or edits a shell's configuration. A release is a tag pushed by hand on a commit
the release workflow accepts. Nothing in the tree carries the version: `go
install` reports it from the module info and a checkout reports `devel`.
`spooler --version` is the `version` command, as `git --version` is `git
version`, the root's one flag not about the connection; it has no environment
twin, since exported it would replace every command.

## The skill

`skills/spooler-cli/SKILL.md` is an [Agent Skill](https://agentskills.io) for
agents that *use* the CLI, mirrored into `spoolersh/ai`. Published content,
like the README; not loaded by agents working on this repo, and says nothing
about how the CLI is built.

- **Whoever changes the CLI's contract updates the skill in the same change**:
  a command, a flag, an output shape, an exit code, an environment twin, a
  rule in the Commands section. Nothing checks this mechanically: before a
  hand-off, read the skill against `spooler help` and each command's `-h`
  and fix every difference.
- It never restates API behaviour beyond what a script needs in context
  (what a settle does, when a send retry is safe), copied from the docs and
  cited.
- Its `metadata.version` is the tag it describes, verbatim: a release commit
  bumps it, and the release workflow refuses a tag that disagrees.
- The voice of the README and the help texts: terse, factual, the docs'
  vocabulary, flags as the help prints them.

## Editing files

Agents change files only with the edit tool, one visible edit at a time; never
with a script (`sed -i`, awk, python, heredocs into source). Scripts that only
read are fine.

## Commits

Agents never commit, tag or push; the working tree is the hand-off. Commit
messages follow the Go convention "where: imperative what".
