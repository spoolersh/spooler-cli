---
name: spooler-cli
description: Use the spooler command line tool to work with the Spooler message queue from a shell, a script or an agent. Covers install, connection flags and SPOOLER_* environment variables, every command (send, recv, ack, nack, release, fail, renew, inspect, recover, discard, messages, queues, stats), lease files, output formats (-o json, table, text, csv), exit codes and script patterns. Use when a task runs the spooler binary, mentions spooler-cli, SPOOLER_API_KEY, a lease file, or needs to send, receive or triage Spooler messages without writing code.
metadata:
  author: spoolersh
  version: v0.2.1
  docs: https://github.com/spoolersh/spooler-cli
---

# spooler CLI

Binary `spooler`, repo https://github.com/spoolersh/spooler-cli, open source, a
thin wrapper over the [Go SDK](https://github.com/spoolersh/spooler-go).

Built for scripts and agents: data on stdout, errors on stderr, documented exit
codes, `-o json` as a stable contract. `spooler help <command>` or `spooler
<command> -h` print the full flag list; every command's help ends with its docs
URL. The delivery contract is https://docs.spooler.sh/delivery; the `spooler`
skill explains it in full.

## Rules

- A receive leases; settle every delivery with `ack`, `nack`, `release`, `fail`
  or `discard`, or the lease times out and the message is redelivered. Delivery
  is at least once: a script's effect must be idempotent.
- `nack` is the error path (counts a retry); `release` is for "not the
  message's fault" and counts nothing; `fail` marks poison for a human. Both
  bullets follow https://docs.spooler.sh/delivery, which rules.
- A lease or handle is an opaque credential: present it back as is. Prefer
  `--lease-file` over `--lease`, which is visible in `ps`.
- The CLI adds no guarantee and hides none: it never acks or retries on its own
  beyond the SDK's transient retries, and never prompts.

## Install

From source, whenever a Go toolchain is on the machine:
```sh
go install github.com/spoolersh/spooler-cli/cmd/spooler@latest
```

From GitHub releases (puts `spooler` in ~/.local/bin):
```sh
curl -fsSL https://spooler.sh/install | sh
```

## Connection

Flags go before the command; each has an environment twin, and the flag wins:

| Flag | Env | Meaning |
| --- | --- | --- |
| `--api-key` | `SPOOLER_API_KEY` | required; prefer the variable (not in `ps` or history) |
| `--host` | `SPOOLER_HOST` | `host[:port]`; default `api.spooler.sh` |
| `--no-tls` | `SPOOLER_NO_TLS` | no TLS, the key travels in plain text; local image only |
| `--spool`, `-s` | `SPOOLER_SPOOL` | required on every command that calls the API; no spool is assumed, not even `default` |
| `--queue`, `-q` | `SPOOLER_QUEUE` | required where the operation names a queue |
| `--profile`, `-p` | `SPOOLER_PROFILE` | profile from `~/.config/spooler/config.yaml` (`--config`, `-C`) |

Local image, free for development and testing (any key, one spool `default`,
nothing persists):

```sh
docker run --rm -p 127.0.0.1:8080:8080 spoolersh/memspoold
export SPOOLER_HOST=127.0.0.1:8080 SPOOLER_NO_TLS=true SPOOLER_API_KEY=local SPOOLER_SPOOL=default
```

Profile file:

```yaml
profiles:
  local:
    host: 127.0.0.1:8080
    no-tls: true
    api-key: local
    spool: default
```

Precedence: flag > profile > environment variable.

## Commands

```
spooler send     --queue Q [--delay D] [--dedup-string K | --dedup-hash H] [--return-handle] (--data S | --data-file F)
spooler recv     --queue Q [--wait D] (--ack | --lease-file F | -i) [--data-file F] [--overwrite]
spooler ack      (--lease L | --lease-file F)
spooler nack     (--lease L | --lease-file F)
spooler release  (--lease L | --lease-file F) [--delay D]
spooler fail     (--lease L | --lease-file F)
spooler renew    (--lease L | --lease-file F)
spooler inspect  --handle H [-i] [--meta-file F] [--data-file F]
spooler recover  --handle H [--delay D]
spooler discard  (--handle H | --lease L | --lease-file F) --yes
spooler messages --queue Q [--state S] [--return-handle] [--limit N]
spooler queues list [--match RE] [--limit N]
spooler queues describe --queue Q
spooler queues create  (--queue Q | --file F) [settings flags]
spooler queues update  (--queue Q | --file F) [--uid U] [settings flags]
spooler queues delete  --queue Q --mode force|drain [--uid U] --yes
spooler stats
spooler version
```

Prefer using long flag names for readability.

Settings flags: `--lease-timeout D`, `--max-retries N`, `--dedup-window D`,
`--retention D`, `--recv-rate-limit R --recv-burst N` (`--recv-rate-limit
unlimited` removes the limit on update). Omitted means the server default
on create and the current value on update.

Durations take units: `20s`, `5m`, `1h30m`, `1d`; a bare number is seconds.
Prefer units for readability. Values that are not whole seconds are a usage
error.

Rules that catch scripts:

- `send` needs `--data S` or `--data-file F` (`-` is stdin). Never reads stdin
  unasked. `--data ""` sends an empty message.
- `recv` needs at least one of `--ack` (settle after the payload is written),
  `--lease-file F` (keep the lease as JSON, mode 0600) or `--include-meta`/`-i`
  (print attributes before the payload, an empty line between). A bare `recv`
  is a usage error: a lease nobody settles would park the message.
- `recv` with nothing to receive exits 1 with empty stdout.
- `discard` and `queues delete` require `--yes`; nothing ever prompts.
- A lease file is any JSON object with a `lease` member; `ack`, `nack`,
  `release`, `fail`, `renew` read it, none deletes it.
- Retrying a `send` after exit 100 is safe only with the same `--dedup-string`
  or `--dedup-hash` the first attempt carried, inside the queue's dedup window
  (be conservative in your estimation on the window size).
- An output flag is refused where it would do nothing, exit 2: `ack -o json` is
  an unknown flag (so are `nack`, `release`, `fail`, `discard`, `recover`,
  `queues update` and `queues delete` with one); `recv` and `inspect` take
  `-o`, `--columns`, `--template`, `--human` or `--compact` only with `-i`;
  `--columns` with `-o json`, `--human` with `json` or `csv`, and `--template`
  with `-o` or `--columns` are usage errors.

## Output

`-o json` (default on records) prints one indented document per result;
`--compact` makes it one line each (NDJSON).

`-o table` or `-o csv` are for listings only, default is `table`.

`-o text` is `key: value` lines.

`--human` turns times into ages and seconds into durations on `text` and
`table`.

`--columns a,b` selects fields (globs: `stats.*`)

`--template '{{ .id }}'` renders each result through a Go `text/template`;
excludes `-o` and `--columns`.

Field names are the API's JSON names (`id`, `lease`, `leaseExpiresAt`, `state`,
`handle`, `stats.visible`).

`recv` and `inspect` write the payload byte for byte to stdout or `--data-file F`.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | OK |
| 1 | Nothing received within the wait |
| 2 | Usage error |
| 4 | Failed; do not retry |
| 8 | Retry later (transient; the SDK already retried) |
| 100 | Result unknown: the change may have happened |
| 128+N | Interrupted by signal N |

## Script patterns

One message, processed by a command, acked on success. `--overwrite` replaces
the files a previous run left; without it an existing one is a usage error:

```sh
spooler recv --queue jobs --wait 20s --lease-file lease.json --data-file payload.bin --overwrite
code=$?
case $code in
  0) ;;              # received a message
  1) exit 0 ;;       # nothing to receive within the wait
  *) exit "$code" ;; # 4 do not retry, 8 retry later, 100 unknown
esac
if ./process payload.bin; then
  spooler ack --lease-file lease.json
else
  spooler nack --lease-file lease.json
fi
```

Send a file with a dedup key:

```sh
spooler send --queue orders --dedup-string "order-$ID" --data-file order.json --compact
```

Triage failed messages; `--data-file /dev/null` prints the attributes alone:

```sh
spooler messages --queue orders --state failed --return-handle -o json --compact \
  | jq -r .handle | while read -r h; do
    spooler inspect --handle "$h" -i --data-file /dev/null
    spooler recover --handle "$h"          # or: spooler discard --handle "$h" --yes
  done
```

Describe a queue, then recreate it elsewhere:

```sh
spooler queues describe --queue orders -o json > orders.json
spooler --profile staging queues create --file orders.json
```

Wait for the local image in CI: poll `spooler queues list` until exit 0.
