# Agent instructions

## What this is

The `spooler` command: a CLI for Spooler, a hosted message queue. Repo
`github.com/spoolersh/spooler-cli`, binary `spooler`, built from
`cmd/spooler` (a root `main` would install as `spooler-cli`). It is a thin
wrapper over the Go SDK, `github.com/spoolersh/spooler-go`: every command is
SDK calls plus input parsing and output rendering, nothing else.

It serves humans and AI agents equally. Both get the same commands, the
same output and the same exit codes; nothing switches format behind the
caller's back.

Priorities: correctness > predictability > ergonomics > features. The
CLI adds no guarantee and hides none: it never acks, renews or settles on
the caller's behalf unless a flag explicitly asks, and adds no retry of
its own. The SDK's retries stay at their default: they cover only the
conditions the docs class as transient and as having changed nothing, so
a rate limit that clears in a second does not fail a command, and an
unknown result is returned at once, since the CLI never sets
`RetryUnknown`. Exit 8 therefore means the condition outlasted the SDK's
attempts.

## North stars: git and curl

When a design question is open and this file is silent, do what git
does; for connection and compatibility, what curl does. Where they and
this file differ, this file wins.

- **git:** options that say where and as whom it runs go before the
  command (`git -C dir`); everything else belongs to the command that
  reads it and follows its name (`git log --oneline`). Here: `spooler
  --host H --api-key K queues list --spool S -o json`.
- **git:** machine output is a separate, stable contract (`--porcelain`);
  here it is `-o json`, and the `text` format may improve freely.
- **git:** `git help <command>` and `<command> -h` give the same
  per-command help. It never prompts where a flag can say yes (`git clean
  -f`); here, `--yes`.
- **go:** the shape of an error line, and of the reply to an unknown
  command: `go buidl: unknown command` then `Run 'go help' for usage.`,
  two lines and no guess at what was meant. With a command list that
  fits one screen, the help it points at is the suggestion; git's "did
  you mean" earns its place at git's hundred and fifty commands, not
  here.
- **curl:** a released flag, output or exit code keeps working forever;
  every exit code is documented; data goes to stdout, diagnostics only
  when asked.

## Where truth lives

- **The public docs at https://docs.spooler.sh are the contract** for
  what the API does: operations, kinds, statuses, limits, retry rules.
  This file never restates API behaviour; when a task needs a fact about
  the API, read the docs page and cite it. When docs and the evaluation
  server disagree or the docs are silent, ask the project owner.
- **The SDK is the only client.** No HTTP from this repo: if a command
  needs something the SDK lacks, it is an SDK change first. The SDK
  hides the transport, and so does the CLI: no URLs, status codes or
  other HTTP notions in its flags or output, so a change of transport
  goes unnoticed. The rule covers the CLI's own words; an SDK message is
  passed through whatever it names, as the Errors section says, so a
  transport failure may read `Post "https://...": connection refused`,
  which is the SDK speaking to the person debugging it, as curl names
  the host it could not reach.
- **This CLI's own contract** -- command names, flags, output shapes, exit
  codes -- is this file plus the README, and is public from v0.1: a change
  to any of them is a breaking change.

## This repo is public

Everything here is a client of the documented API and nothing else. Never
let in: internal hostnames, topology, replication or storage details,
limits typed from memory, pricing, roadmap, or anything from the private
repos' source. Every statement must be checkable with an API key.

The local evaluation image `spoolersh/memspoold` is NOT open source: it
ships under the Spooler Local Evaluation License. Never call it "OSS" or
"open source"; say "free for development and testing". The CLI itself is
open source.

## Vocabulary

Every command, flag and output field uses the docs' words; the CLI coins
none. Grep the docs before naming anything.

- Verbs mirror the API's operations: `send`, `recv` (alias `receive`),
  `ack`, `nack`, `release`, `fail`, `renew`, `inspect`, `recover`,
  `discard`. `list` (alias `ls`) and `describe` for reads.
- "Failed" is a message state, not a noun: failed messages are
  `messages --queue q --state failed`.
- Sign in/up/out, never login/logout. `spooler login` is not a command,
  not even an alias: it is unknown, and the reply points at the help.
- Flags mirror the request parameters they set: `--delay` (delay seconds),
  `--dedup-string`, `--uid` (the precondition on update and delete),
  `--wait`.
- `-o json` field names are the API's JSON names (`id`, `lease`,
  `leaseExpiresAt`, `retries`, `state`, `handle`, ...), so a reader of the
  docs can predict the output without reading the CLI's.

## Commands (v1: data plane only)

```
spooler [--host H] [--no-tls] [--api-key K] <command> ...

spooler send     --queue Q [--spool S] [--delay D] [--dedup-string K | --dedup-hash H] [--return-handle]
                           (--data S | --data-file F)
spooler recv     --queue Q [--spool S] [--wait D] (--ack | --lease-file F | -i) [--data-file F] [--overwrite]
spooler ack      --lease L | --lease-file F [--spool S]
spooler nack     --lease L | --lease-file F [--spool S]
spooler release  --lease L | --lease-file F [--spool S] [--delay D]
spooler fail     --lease L | --lease-file F [--spool S]
spooler renew    --lease L | --lease-file F [--spool S]
spooler queues list [--spool S] [--limit N] [--columns C]
spooler queues describe --queue Q [--spool S]
spooler queues create  --queue Q | --file F [--spool S] [settings flags]
spooler queues update  --queue Q | --file F [--spool S] [--uid U] [settings flags]

settings flags: [--lease-timeout D] [--max-retries N] [--dedup-window D]
                [--retention D] [--recv-rate-limit R --recv-burst N]
spooler queues delete  --queue Q --mode force|drain [--spool S] [--uid U] --yes
spooler messages --queue Q [--spool S] [--state S] [--limit N] [--columns C]
spooler inspect  --handle H [--spool S] [-i] [--meta-file F] [--data-file F] [--overwrite]
spooler recover  --handle H [--spool S] [--delay D]
spooler discard  (--handle H | --lease L | --lease-file F) [--spool S] --yes
spooler stats    [--spool S]
spooler version
```

Connection flags (`--host`, `--no-tls`, `--api-key`) are the root's and
go before the command; every command that prints a result takes
`--output`/`-o` and `--template`, and a command that prints nothing on
success (`ack`, `nack`, `release`, `fail`, `discard`, `recover`, `queues
update`, `queues delete`) takes neither, since a flag that silently does
nothing is what this file forbids: `ack -o json` is an unknown flag.
Queue settings flags are the fields of the docs'
create/update request schemas, nothing invented, named without the unit
suffix, as `--delay` stands for `delaySeconds`: `--lease-timeout 5m` for
`leaseTimeoutSeconds`, a duration; `--recv-rate-limit 100/s` with
`--recv-burst N` for `recvRateLimit`, both or neither, the rate in the
human form since the API's `intervalMicros` is not one a person writes,
and `--recv-rate-limit unlimited` removes the limit, the word `--human`
prints for none. On `update` a setting no flag or file gives keeps its
current value, and nothing to set is a usage error. `--file F` (`-f`, as kubectl) reads a queue
record as `queues describe -o json` prints it, so a described queue can
be created again: its name and settings are taken, its uid, state and
stats ignored, and a flag given overrides a value in it. A record's zero
`retentionSeconds` is a retention never set, which the API refuses below
60, so it is read as absent and the server applies its default; every
other zero setting means what the docs say it means and is sent.

Decided rules:

- **Everything is an explicit flag.** The queue is `--queue`, required
  where the operation names a queue; a lease is `--lease`, a handle
  `--handle`. No positional operands for API objects: one form in docs,
  scripts and transcripts.
- **Payload:** `--data S` or `--data-file F`, where `-` is stdin; neither
  or both is a usage error naming the three forms. Never stdin unasked:
  a cron job's stdin is `/dev/null`, and the API appends an empty
  message for an empty body, so an omitted flag would enqueue silently.
  An empty message is `--data ""`. Two flags rather than curl's `--data
  @file`: a payload is arbitrary bytes and may start with `@`, which is
  why curl needed `--data-raw`. A file that is not regular (a pipe,
  `<(cmd)`, `/dev/stdin`) is streamed once without a length and cannot
  be retried; a regular file is seekable and can.
- **Dedup key:** `--dedup-string` and `--dedup-hash` exclude each other.
  Presence is tracked, not the value, as for `--data`: a given but empty
  one goes to the API as the empty key, which it refuses, so
  `--dedup-string "$ID"` with `ID` unset cannot pass as a send without a
  key.
- **"Was this flag given" is answered only by a tracking variable.** A
  flag whose presence the code acts on, to choose a path, refuse a pair or
  require one of a set, is defined through `flagvars.OptVar` (a pointer,
  nil when omitted) or `flagvars.Opt` (a `given` bool), and is never read
  back through its zero value: an empty string is a value the caller
  passed, not an omitted flag. So `ack --lease "$LEASE" --lease-file F`
  with `LEASE` unset is refused as the pair it is, and `ack --lease
  "$LEASE"` alone goes to the API as the empty lease; neither acts on a
  lease the caller did not name, and an empty path goes to the OS, which
  refuses it before any request. What an empty value means is the value's type's business,
  decided where the type is parsed, never the flag's.
- **The CLI maps flag presence onto the SDK's optional types and never
  checks a value's format.** A flag has three states: omitted, given
  empty, given with a value. Where the SDK field is a pointer or a tagged
  union (`QueueUID *string`, `Dedup`), the three map one to one: an
  omitted `--uid` is a nil `QueueUID`, a given one is `&value`, empty
  included, and the server refuses the empty one. Where the SDK field is
  bare, its zero is absent on the server as well, so given-empty and
  omitted mean the same thing and nothing is lost. The CLI therefore has
  no empty-value rules of its own: `--uid ""`, `--dedup-string ""` and
  `--lease ""` all go out and come back refused. What it never checks is
  a value's format: uid length or alphabet, lease shape, hash hex.

  **The test for whether a check belongs in the CLI: could it move into
  the SDK without the CLI changing? If yes, it is validation on behalf of
  the SDK and belongs there. If no, because the SDK cannot observe the
  condition, it is translation and belongs here.**

  With the SDK's optional fields shaped as above, nothing is left on the
  CLI's side of that line but presence itself.
- **Lease:** `recv` takes `--ack`, `--lease-file F`, `--include-meta`/`-i`
  or any of them together; none is a usage error naming the three. A
  receive leases the message, and on a queue without a lease timeout a
  lease nobody settles parks it as pending, so an omitted flag must not
  lease silently: the caller says whether the lease is settled, kept or
  printed, as `send` says where the payload is. The check runs before the
  receive.
- **Shorthands** exist only for flags a person types often, and follow one
  rule for twins: where a flag takes a value and another takes a file
  holding that value, read or written, the value's letter is lowercase and
  the file's is the same letter in uppercase: `-l` `--lease` and `-L`
  `--lease-file`, `-d` `--data` and `-D` `--data-file`. A flag without a
  twin takes the letter its precedent uses: `-f` `--file` as kubectl, `-C`
  `--config` as git, `-n` `--limit` as `head` and `git log`. The rest:
  `-s` `--spool`, `-q` `--queue`, `-o` `--output`, `-c` `--columns`, `-i`
  `--include-meta`, `-p` `--profile`.
  `--handle` has none: `-h` is help, and a handle is pasted, not typed. A
  shorthand is a released contract, so one is added only on evidence of
  use and never removed; consent flags (`--yes`, `--ack`, `--overwrite`)
  never get one, their length being part of the consent.
- **Listings** walk every page through the CLI's own pager, so `queues
  list` and `messages` print every item unless `--limit N` (`-n`) says to
  stop after N, in which case no page beyond the need is fetched. 0, the
  default, is every item: a listing is never cut silently, as `git log`
  and `kubectl get` are not. The page size is the server's default, never
  one the CLI asks for: the docs bound `limit` and say nothing about a
  request beyond the bound. A cursor to resume from is not exposed yet.
- **Spool:** `--spool`, else `SPOOLER_SPOOL`, else the profile's; none is
  refused by the SDK as an invalid request, exit 2. No spool is assumed,
  not even one named `default`.
- **Durations** are written with units (`--wait 20s`, `--retention 1d`,
  `--lease-timeout 5m`), in the forms `internal/human` reads: `d`, `h`,
  `m`, `s` and below, in any combination (`1h30m`); a bare number is
  seconds (`--retention 86400`), as `sleep`, `timeout`, gcloud and systemd
  read one and as the docs' `*Seconds` fields are written. The API takes
  whole seconds, so a value that is not whole seconds is a usage error,
  never rounded. The help names the value `duration`. Omitted means the
  server's default.
- **Destructive operations** (`queues delete`, `discard`) require `--yes`;
  without it they fail with exit 2 and say why. Never prompt.

Later, in order: `consume --queue Q -- cmd` (payload on the child's stdin,
metadata in its environment, ack on exit 0, nack otherwise; ships only when
its renew/signal/ack handling is proven correct -- a wrong ack loses a
customer's message); `ack-and-send`; `sign-in`/`sign-out` (device flow,
account-level access) with a config file and profiles; account-plane
commands (keys, spools). Never: `tail` (it would lease messages while
sounding read-only), more output syntaxes (`--output yaml`), telemetry,
update-available nags, colour themes (every colour is a palette slot the
terminal's theme resolves; no fixed value, no option to pick one).

## Credentials

The connection mirrors the SDK's `Client`: `--host` (`host[:port]`, the
SDK's default when unset), `--no-tls` (the SDK's `InsecureDisableTLS`:
no TLS, the key travels in plain text; for the local evaluation image
only) and `--api-key`. The `--api-key` help says it is visible in `ps`
and shell history, and the docs show the environment. A profile in the
config file (`--config`, `-C`, default `~/.config/spooler/config.yaml`;
`--profile`, `-p`) may hold the connection too. A connection value is
taken from the flag, else the profile, else the variable: the flag parsers
run in that order and a later one leaves set what an earlier one set, so a
flag beats the profile and the profile beats a variable left exported in
the shell, which is the stale `SPOOLER_API_KEY` beside `--profile prod`
that would otherwise win silently, as it does in the AWS CLI. The north
stars put the environment above a config file; here a profile is named
on the command line and a variable is ambient, so the named one wins. The
profile's own selection, `--profile` and `--config`, is read from the
environment before the profile, the rest after it, which is why the
environment parser is listed twice. The parser runs on every flag set and
gives each the profile values it defines, the connection at the root and
the spool on a command, so the spool follows the same order. Help never
shows a value a profile or a variable supplies, as git's and kubectl's do
not: the help is the same whatever the shell holds. A missing key is
refused by the SDK on the first call ("api key is required", exit 2 as
an invalid request), and `--help` names the two sources; no hidden
default key, not even for localhost, and nothing is inferred from the
host: `localhost` without `--no-tls` still uses TLS.

**The context flags have an environment twin**, and no other flag does.
The context is where, as whom and how loudly the CLI runs, and the spool
and queue a session works in: `--config`, `--profile`, `--host`,
`--api-key`, `--no-tls`, `--log-level`, `--spool` and `--queue`, eight
flags, the allowlist in `flagParseOptions`. The twin is derived
mechanically, `SPOOLER_` plus the flag name in upper case with `-` as `_`
(`SPOOLER_HOST`, `SPOOLER_NO_TLS`, `SPOOLER_API_KEY`, which the SDK
dictates; `SPOOLER_QUEUE`), and `--help` shows it next to the flag. The
flag wins over the variable. This is what git, curl and kubectl do:
`GIT_DIR` and `GIT_AUTHOR_NAME`, the proxy variables and `CURL_CA_BUNDLE`,
`KUBECONFIG`, and no variable for any option of a command; the AWS CLI's
`AWS_DEFAULT_OUTPUT` is the exception and the complaint. Everything else
shapes one invocation and is asked for on its command line: consent
(`--yes`, `--ack`, `--overwrite`) is given per command, never exported;
an output choice (`--output`, `--columns`, `--template`, `--compact`,
`--human`) is refused by the commands it does not fit, so an exported one
would break every such command for the sake of the others; `--version`
exported would replace every command; `--debug` dumps payloads to stderr,
which is asked for per invocation as consent is, never left in a shell
for every later command to obey; a lease or a handle is per message,
and an exported `--lease-file` would make `recv` lease on the strength of
the environment; `--wait`, `--delay`, `--limit` and the rest are
preferences, which a shell alias holds. `--no-color` has no twin since
`NO_COLOR`, the convention every tool reads, does the same.

## Output

**stdout carries data; stderr carries errors only.** Nothing else ever goes
to stderr: no progress, hints or metadata (diagnostics only when a debug
flag explicitly asks). The two diagnostics are the root's: `--log-level`
prints the SDK's trace as JSON lines with fixed keys, a stream a machine
reads, and `--debug` points the SDK's dump of every request and response,
payloads included, at stderr in whatever form the SDK gives it, which may
change with the transport and is for a person's eyes. They are separate
flags because one is readable and the other is anything; no level of the
first turns on the second. The dump goes to stderr only: a `--debug-file`
waits for a need.

**The destination changes colour and nothing else.** Whether stdout is a
terminal, a pipe or a file decides only whether escape sequences are
written; the bytes between them, the shape, the indentation and the
fields are the same, so `spooler X | cat` and `| tee` show what `spooler
X` showed, and a script reads what a person saw. The per-format rules
below are instances of this one.

The format is `--output`/`-o`: `text` and `json` on every command,
`table` and `csv` on list commands; any other value is a usage error. The
default is `table` on list commands and `json` elsewhere, as gcloud does
(`list` a table, `describe` a structured record); a value means the same
on every command. A record's default is `json` so that a script and a
person get the same bytes unflagged, indented either way and coloured
only at a terminal. All four are mechanical renderings of one output struct through
one field walker, with no templates and no second syntax, which is what
keeps them from being the matrix the Commands section forbids. Templates
are `--template`, never `--format` (docker's format selector, so
`--format json` would be tried here) and never a value of `-o` (kubectl's
`-o go-template=...` quoting). No `--json` alias: one form in docs and
transcripts.

- **`text`**: one record per value: a `key: value` line per field, the
  key its `-o json` name with nested ones dotted (`stats.visible: 12`),
  then an empty line. A field `json` would omit has no line. No alignment.
  On a terminal the keys are coloured bright blue from the terminal's
  palette, and the path of a nested key (`stats.` of `stats.visible`) is
  the same colour faint, so the groups read as groups while the line stays
  flat; off a terminal, with `NO_COLOR` set or with `--no-color`, there is
  no colour and nothing else differs. Colours are palette slots, never
  fixed xterm-256 values, as git colours its output: only the user's theme
  knows whether the background is dark or light. The empty
  line is the contract and the lines may grow; a reader takes everything
  up to the first empty line as one record.
- **`table`**: one row per value under a header row of the names in upper
  case with `_` (`STATS_VISIBLE`); an omitted field prints `-`. Off a
  terminal it is the same format without colour or truncation; `NO_COLOR`
  and `--no-color` are honoured. It never changes shape because stdout is
  a pipe.
- **`json`**: one JSON document per result, each indented, as jq prints
  them, whatever the destination; a list is those documents one after
  another, which jq and every JSON decoder read as a stream. `--compact`
  is `jq -c`: one document per line, NDJSON, for `head`, `grep`, `wc -l`
  and `while read`; with any other format or with `--template` it is a
  usage error. It is a contract: fields are added, never renamed or
  removed. Only colour follows the terminal: keys bright blue as
  `text`'s and string values yellow, palette colours under the same
  `NO_COLOR` and `--no-color` rules as `text`, and none in a pipe.
- **`csv`**: RFC 4180, a header record of the `-o json` names dotted, an
  omitted field empty.
- **`--columns`** on `text`, `table` and `csv` selects and orders the
  fields: a comma-separated list of `-o json` field names, nested ones
  dotted (`name,state,stats.visible`), so a key copied from `text` output
  is a valid value. Each is a glob pattern (`github.com/gobwas/glob`, no
  separators: `*` crosses dots) expanding to its matches in field order:
  `stats.*`, `*Total`, `stats.{visible,pending}`, `*`; a field selected
  twice keeps its first position. A pattern matching no field is a usage
  error that lists the fields; so is `--columns` with `-o json`.
- **`--template T`** renders each result through a Go `text/template` and
  is its own output mode: with `-o` or `--columns` it is a usage error.
  The data is the output struct as nested maps by its `-o json` names,
  through the field walker every format renders through, so `{{.id}}`
  and `{{.stats.visible}}` use the docs' names; each field is its Go
  value as is, so a number prints as one and compares as one (`{{if gt
  .retries 2}}`, false at zero) and a time is a `time.Time` with its
  methods (`{{.createdAt.Format "2006-01-02"}}`). A field the value
  omits by the json rule (`handle` without `--return-handle`) is nil:
  `<no value>` printed, false in a condition, so a template over an
  optional field writes `{{if .handle}}`. Every name of the type is a
  key, so the one key the data can lack is a name that is no field, and
  that is an execution error naming the fields, as `--columns` names
  them; text/template's default would print `<no value>` for a typo at
  exit 0. One execution per result with a newline appended, as docker
  does; lists stream one line per item. A parse error is exit 2, an
  execution error exit 4, the template's message passed through. No
  functions beyond `text/template`'s own; any added later are permanent.
- **`recv` and `inspect`** write the payload to stdout byte for byte,
  nothing escaped, on a terminal too, or to `--data-file F`. The metadata, `recv`'s lease record or `inspect`'s attributes,
  goes only where a flag sends it, and the format follows the flag, never
  the destination, as it does in every CLI this file leans on:
  `--include-meta`/`-i` prints it on stdout ahead of the payload in the
  `--output` format, `text` or `json` (`table` and `csv` are usage errors
  here), as `curl -i` prints headers; `--lease-file F` and `--meta-file F`
  write it as JSON to F, `/dev/stdout` included, as `curl -D` dumps
  headers. `-o`, `--columns`, `--template`, `--human` or `--compact`
  without `-i` is a usage error naming `-i`, since a flag that silently
  does nothing is what this file forbids; `-i` together with a `-file` flag that is stdout is one too,
  since two records on one stream would leave a reader no way to find the
  payload. Without any of them `inspect` prints the payload alone, as
  `curl` prints a body; `recv` without any needs `--ack`, and the lease is
  settled and nothing is kept. One rule covers every destination: where a
  record and the payload share a stream, the record leads and one empty
  line separates them, so a reader takes everything up to the first empty
  line as the metadata, as HTTP and `curl -i` are read. So `recv -i` and
  `inspect -i` are the record, an empty line, the payload; with
  `--data-file F` added the record is alone on stdout; `recv --ack` and
  bare `inspect` are the payload alone, so `inspect --handle H | spooler
  send --queue Q --data-file -` moves a payload as it is; and `--lease-file
  F --data-file F` is one file in the same shape with the JSON record
  leading. Nothing
  received is exit 1 and empty stdout. With `--ack`, the ack is sent only
  after the payload is fully written; a failed write releases the lease
  and the error says so, and a `--lease-file` or `-i` named alongside
  `--ack` is still honoured. A closed stdout is such a write error: `main`
  asks for SIGPIPE, so the write fails with EPIPE and reaches the release,
  where the signal's default would end the process first.
- **Lease file**: `recv` writes one JSON object with the
  receive's attributes under their docs' names (`id`, `retries`,
  `createdAt`, `lease`, `leaseExpiresAt`, absent when the lease never
  expires, and `dataSize`,
  the payload's bytes as the server declared them, absent when it did
  not), mode
  0600, before the first payload byte is read. A reader (`ack`, `nack`,
  `release`, `fail`, `renew`) takes any JSON object with a `lease` member
  and ignores the rest. `renew` leaves the file as it is, since the lease
  token does not change, and prints the new `leaseExpiresAt` as its result;
  a stale expiry in the file is harmless, no reader uses it. A path under `/dev` or `/proc`
  (`/dev/stderr`, `/dev/fd/3`), or one that exists but is not a regular
  file (a pipe), is a stream and is written as is, whatever the shell
  redirected it to. An existing regular file is a usage error unless
  `--overwrite`: the check runs before the receive, so refusing never
  costs a lease, and the new content is written to a file created beside
  it, with the flag's mode, and renamed over it only once the payload is
  delivered too, so nothing received, and a delivery that fails and
  releases the lease, leave the old file as it was; the record is still
  written before the first payload byte is read, and a process killed
  mid-payload leaves it in that file beside the destination. A file the
  receive created is removed when the payload cannot be delivered and the
  lease is released, so no half-written file is left behind. When that
  release fails or ends unknown the lease may be live, so the record is kept
  instead: the created file stays and a replacement lands, and the exit
  is 100 for an unknown release, since whether the lease is held is the
  one thing in doubt; a failed release keeps the delivery's own code,
  with the lease named in the message either way. Once delivered, the
  files are the caller's, and an ack that fails or ends unknown leaves
  the lease file, with the lease in it, in place. No settle
  verb deletes the file. A path's symlinks are resolved first, as the OS
  resolves them, before any `..` is collapsed (`view/../bundle` with
  `view` a symlink is not `bundle`), so a replacement lands on the file a
  link points at and the link stays a link, and one file named by two
  flags under two
  spellings (a symlink, a relative and an absolute path, a case an
  insensitive filesystem ignores) is one destination with the record ahead
  of the payload: by the resolved path where spelling can tell, by stat
  identity where it cannot. The file stdout is redirected to is stdout
  under any path (`--lease-file bundle > bundle` is `--lease-file -`), never
  a file to replace: a replacement would unlink what stdout wrote. A file
  with other hard links is not replaced at all,
  with or without a second flag naming it: a rename replaces one name and
  the others would keep the old content, so `--overwrite` on it is a usage
  error before the receive. Messages keep the path as the caller spelled
  it. The `--*-file` flags are one family with one
  mechanism, and the prefix names the content: `--data-file` the payload,
  `--lease-file` a record holding a credential, mode 0600, `--meta-file`
  (`inspect`) a record holding none, mode 0666, `--file` (`queues
  create` and `update`) a queue record read, never written. No command takes both
  record flags, so mistaking one for the other is an unknown flag, never
  a lease in the wrong file.
- **`--human`** renders, on `table` and `text` only, the fields that have a
  human form: a time as the duration between it and now, past or future
  alike (`createdAt: 3h`, `leaseExpiresAt: 29m`), a seconds count as a
  duration (`30s`, `7d`, at most two units, `1h1m1s` printed `1h1m`), a byte
  count in binary units (`1.5MiB`) and a receive rate limit's interval as a
  count per period (`100/s`, `unlimited` for none). The field set, the
  headers and the `--columns` paths do not change: `CREATED_AT` keeps its
  name with an age in the cell, so nothing is renamed as kubectl's `AGE`
  is. With `json`, `csv` or `--template` it is a usage error, since those
  are contracts. Never a default, even at a terminal: the table is the same
  on and off one. Which fields have a form is a `human` struct tag on the
  output types (`age`, `seconds`, `rate`, `bytes`), nothing inferred from
  names. The forms are `internal/human`'s, the same package the server
  uses, whose parsers read every form back.
- **Errors**: human text on stderr, whatever `--output` is; there is no
  JSON error object. An API error's message is passed through as the SDK
  reports it, never reworded, and the exit code says what to do next. A
  usage error with a reason (`cli.Exitf(2, ...)`) prints its sentence
  alone, as git does for a wrong invocation, since every sentence names
  the flags to pass; a command that needs flags and gets none prints its
  help alone (`new(cli.UsageError)`, no message), since the usage line is
  the answer to a bare invocation. `cli.Usagef` is not used: a sentence
  followed by the help would say everything twice. A leftover argument,
  which the parser stops at so the flags behind it were never read, is a
  usage error naming it (`unexpected argument: "payload"`). Each check
  stands as its own `if` at the top of `Run`, one helper per check; no
  helper bundles several checks. `ensureNoArgs` runs before `noFlags`: a
  stray word hides every flag behind it from the parser, and naming the
  word says why the flags went unread, where the bare usage would not.
- **`--help`** for every command prints a git-style usage line with the
  required flags as `--spool <spool>` and the rest as `[<options>]`, its
  description from the command's `cmd_*.txt` template
  ending with the operation's docs URL
  (`https://docs.spooler.sh/operations/<operationId>`), then the flags
  ordered by meaning as git orders options, not alphabetically: `--spool`
  and `--queue` first, the command's own flags next in alphabetical
  order, the output flags every command shares last. The root lists
  connection, then configuration, then logging. Its help alone opens with
  the wordmark as block characters, 80 columns by 6 rows, at the top of
  the prolog template `main_prolog.txt` (a copy of the brand package's
  `logo.txt`, rendered and kept there; this repo only embeds it), then the
  prolog text before the usage, and ends
  with the exit codes table after the flags, where GNU tools print their
  exit status and man pages their EXIT STATUS; a command's help does not
  repeat any of them. The wordmark is in the terminal's own colour, as the
  rest of the help is: the help is monochrome, as git's, and the
  letterforms are the mark. A command's summary,
  its `Summary()`, appears only in its parent's command list. The prolog and
  epilog are plain text in `main_prolog.txt` and `main_epilog.txt`,
  embedded as the `cmd_*.txt` descriptions are.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | OK |
| 1 | Nothing received (recv found no message within the wait) |
| 2 | Usage error |
| 4 | Failed; do not retry |
| 8 | Retry later (the docs' transient conditions) |
| 100 | Result unknown: the change may have happened |
| 128+N | Interrupted by signal N: 130 on SIGINT, 143 on SIGTERM |

Exit codes say what to do next; the message on stderr says why. Code 8
comes after the SDK's own attempts at a transient condition, so a script
seeing it backs off for longer than the SDK did. Code 100 on `send` is
the one case where a blind retry can duplicate: its error says the message
may have been sent and, as the docs rule, that only a send whose first
attempt carried `--dedup-string` or `--dedup-hash` is safe to retry, with
the same value, within the queue's dedup window; a key added for the
retry protects nothing. Code 100 on `recv` after a failed delivery means
the release of the lease ended unknown: the lease file, when one was
asked for, holds the lease to settle by hand. Neither the SDK nor the CLI retries an unknown result;
whether to is the calling script's decision, made on the code.

0, 1 and 2 follow the GNU utilities (grep: 0 match, 1 none, 2 trouble;
diff likewise). The failure classes are spaced, 4 and 8, so a class can
gain a neighbour later without renumbering, and the unknown result sits
apart at 100 on purpose: it is the one outcome a caller must think about
rather than retry or give up, and a number unlike the others makes it
hard to mistake for one of them. None of the codes collides with a
convention a script may know: not BSD `sysexits.h` (64-78), and not the
shell's own 126, 127 and 128+N; POSIX itself fixes only 0 as success.
Nothing above 125 is returned for a result; an interruption exits with
128 plus the signal's number, the shell's own report of a signalled
process, so a script reads it as it reads any command it killed.

The mapping from errors to codes is one function, `mapExit`, applied
once at the root to whatever a command returns. An error that already
carries a code, a usage error from `cli.Exitf` or `recv`'s silent 1,
keeps it; the rest is checked in order: the SDK's `ErrResultUnknown` ->
100 (first, as the SDK says: a later attempt's answer does not clear an
earlier unknown); `ErrUnavailable` or a kind the docs list as transient
(`rate_limited`, `queue_creating`, `queue_busy`, `dedup_in_flight`) ->
8; `ErrInvalidRequest` -> 2, the SDK's refusal of an empty key among
them; any other API error, a cancelled or expired context and every
local failure (files, a broken pipe, a template that fails to execute)
-> 4, since a timeout says nothing about whether a retry is safe, and
one that struck with the request on the wire carries `ErrResultUnknown`
and is 100; `recv`
with nothing -> 1 with empty stdout and empty stderr, like grep on no
match. After a signal, which the runner reports as the cancellation's
cause (`cli.SignalError`), `mapExit` gives no code, so the runner exits
with 128 plus the signal's number in silence, as an interrupted command
does, with two exceptions it reports as usual, since each says something
the caller must act on: an unknown result, coded by `mapExit` as ever,
and an error that names a live lease, `recv`'s failed ack or release,
which `recv` codes itself through `exitCode` for the runner to trust
first. The root sets the code and prints the message; a command never
prints an error itself, and `send` alone adds to one: a result unknown
says what the docs' retry rule makes of it, that a retry may send the
message twice without a key the first attempt carried, or that one with
the same `--dedup-string` or `--dedup-hash` is safe within the dedup
window.

TODO: a transport failure that reached no server (a refused connection, a
host that does not resolve) is exit 4 today, since the SDK keeps the
transport error as text only; it should be 8, which needs the SDK to mark
it with a condition the CLI can match.

## Libraries and structure

- `github.com/gobwas/cli` for commands: a `cli.Commands` map per noun,
  `cli.Command` per verb. The connection flags are the root
  `cli.Container`'s; the flags every command shares (`--spool`,
  `--output`) are defined into each command's flag set by one helper.
- `github.com/gobwas/flagutil` for parsing: its POSIX parser (`pargs`)
  stops at the first non-option argument, so a command's flags follow its
  name and a connection flag after the command is a usage error.
- `cmd/spooler` parses inputs and maps them onto SDK requests; rendering
  lives apart from command logic so every command renders through the
  same human/JSON writers.
- `github.com/gobwas/glob` for the `--columns` patterns.
- Standard library beyond those, `golang.org/x/*` only if it earns its
  place.

## Coding style

Standard Go, `gofmt`. Comments are concise: what a thing is or does in one
line; a non-obvious why only when the code cannot say it. One field, one
doc comment. Struct literals one field per line. File modes are written
as the standard library writes them, `0600` and `0666`, never `0o600`. Every `fmt.Errorf` and
`cli.Exitf` call has one shape: the format string on its own line, then
the arguments on one line, then the closing parenthesis. A call with no
arguments stays on one line when it fits in 80 columns, a tab counting as
four; a format string that does not fit is split with `+`:

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

In tests, `t.Errorf` and `t.Fatalf` follow the same shape unless the whole
call fits in 80 columns. A flag's help string has the same shape: on its
own line after the name and default, split with `+` when it does not fit:

```go
fs.BoolVar(p,
	"yes", false,
	"confirm a destructive operation; without it nothing is changed,"+
		" and nothing ever prompts",
)
```

A name the caller supplied, interpolated into an error or a help string (a
queue, a spool, a profile, a column, a format, a lease), is `%q`, never
`%s`, so an empty or space-bearing value still reads: `removes the queue
%q for good`. A file path is the exception and is `%s`, as the OS and
`ls` print it. A flag is literal text with its dashes (`pass --yes`), and
a list of names is joined and printed with `%s`. An error that asks for a
flag names the flag alone, never a placeholder for its value: `pass
--lease or --lease-file`, not `--lease L or --lease-file F`; the
placeholders belong to the synopsis and the help, where the value is
described. A flag or command in a message is bare when its dashes delimit
it (`pass --yes`, `takes effect only with --include-meta`) and in
backticks when it has a space inside (`` `-o table` ``, `` `--data-file
-` ``, `` `spooler send` ``), as cli quotes a command path. A command's
summary, its `Summary()`, is a line in a list, as `git help` lists
commands: a capital, a verb in the imperative, no period ("Send a message
to a queue"). Its description, in `cmd_*.txt`, is prose, as a man page's
DESCRIPTION: sentences in the third person, the first opening with the
verb ("Sends a message to a queue with the payload of ..."). Assign a multi-line call on its own line, then
`if err != nil`. Every
exported symbol has a doc comment that is a complete sentence starting
with its name.

## Testing

- Table-driven tests with `testing`; test names are lowercase phrases
  without underscores; case structs hold scenario data only, never mock
  functions. Every case is written one field per line with a `name`,
  never as a one-line literal, so a case reads and diffs like the struct
  it is.
- The SDK is reached through a small interface per command group so tests
  stub it (the `Do*` function-field pattern, a compile-time interface
  check, a default error naming the method).
- Every command is tested end to end in-process: args, env and stdin in;
  stdout, stderr and exit code out, compared exactly. The exit-code table
  and the `-o json` shapes are pinned by these tests.
- Assertions: actual first, then want, never "got":
  `t.Errorf("exit: %d; want %d", act, exp)`. Structs with
  `cmp.Diff(exp, act)`.
- No integration suite against the evaluation image: the SDK's own runs
  the wire protocol against it, and what the CLI adds, flags to requests,
  results to output, exit codes, is what the stubbed tables pin. The glue
  in `main`, the real client reaching the commands, is one path, checked
  by hand against a closed port.
- `go test ./...` and `go vet ./...` clean before every hand-off.

## Docs and README

Tone is mq_overview(7): terse, complete, factual. No "simply", "just",
"seamless", "powerful"; never pitch the transport. The README leads with
the local loop (run the evaluation image, export the connection variables,
create a queue, send, recv, ack) and documents the output shapes and exit
codes before the first tag. The "message queues as simple as Go channels"
tagline belongs to the SDK.

## Releases

Order: server deploy, evaluation image push, SDK tag, CLI tag -- the CLI
never ships behaviour the SDK and server do not have. Distribution:
`script/install.sh` over the release binaries, and `go install`; a
Homebrew tap waits for a need. The script is POSIX `sh`, fetches the
latest release's `checksums.txt` to learn the archive's name, fetches the
archive from that release by its tag, checks it against the sum and
installs the binary alone into `SPOOLER_INSTALL_DIR`, `~/.local/bin` by
default, at the version `SPOOLER_VERSION` names, the latest by default;
those two variables are its whole contract and are as released as a flag.
It never uses `sudo` or edits a shell's configuration, and says when the
directory is not on `PATH`. `script/install_test.go` runs it under `go
test` against a fixture release served by a stub `curl` on `PATH`, so the
tests add no variable to the contract; CI also lints it with shellcheck
and `dash -n`. The README carries its raw GitHub URL until the site serves
it under `spooler.sh`. A
release is a tag pushed by hand, `v0.1.0`, on a commit whose `go.mod`
requires a tagged SDK with no `replace`; the release workflow checks that
and runs GoReleaser (`.goreleaser.yaml`), which builds the binaries with
the tag's version written into `main.version` and puts the archives and
their checksums on the release page. The
version is the tag's and nothing in the tree carries it, so a `go install`
reports the same string from the module info; a build from a checkout
reports `devel`. The SDK differs: its version is a constant its own
workflow checks against the tag.
`spooler version` prints the CLI's version and the SDK's; `spooler
--version` is the same command, as `git --version` is `git version`, since
it is what a person or an agent tries first, and the root's one flag that
is not about the connection. It has no environment twin: exported, it
would replace every command.

## Editing files

Agents change files only with the edit tool, one visible edit at a time;
never with a script (`sed -i`, awk, python, heredocs into source). Scripts
that only read are fine.

## Commits

Agents never commit, tag or push; the working tree is the hand-off.
Commit messages follow the Go convention "where: imperative what".
