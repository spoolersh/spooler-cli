<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo-neon.svg">
    <img src="assets/logo-azure.svg" alt="Spooler" width="600" style="padding-block: 4rem">
  </picture>
</p>

---

The official command line interface for [Spooler](https://docs.spooler.sh).

## Installation

#### Linux or macOS

Using a handy [install.sh](https://raw.githubusercontent.com/spoolersh/spooler-cli/main/script/install.sh):

```sh
curl -fsSL https://spooler.sh/install | sh
```

The script downloads the latest release archive for the machine, checks it
against the release's `checksums.txt` and puts `spooler` in `~/.local/bin`.
`SPOOLER_VERSION=0.1.0` picks a release; `SPOOLER_INSTALL_DIR` another
directory. It script touches nothing else: no `sudo` or shell configuration.

Alternatively, you can download an archive from the **[releases
page](https://github.com/spoolersh/spooler-cli/releases)** by hand.

#### Building from source
```sh
go install github.com/spoolersh/spooler-cli/cmd/spooler@latest
```

## Usage

For detailed usage docs run:
```sh
spooler help
```

You can test `spooler` CLI against [locally running image](https://docs.spooler.sh/local):

```sh
docker run --rm -p 127.0.0.1:8080:8080 spoolersh/memspoold
```

> The image is free for development and testing, has one spool named `default`, and loses its data when it stops:

Configure the CLI:

```sh
export SPOOLER_HOST=127.0.0.1:8080
export SPOOLER_NO_TLS=true
export SPOOLER_API_KEY=local
export SPOOLER_SPOOL=default
```

Create a queue, send a message, receive it, and acknowledge it:

```sh
spooler queues create --queue greetings
spooler send --queue greetings --data 'Hello, Spooler!'
spooler recv --queue greetings --lease-file lease.json
spooler ack --lease-file lease.json
```

- `spooler recv` prints the message and saves its lease in `lease.json`
- `spooler ack` acknowledges the message using that lease

For production, unset `SPOOLER_HOST` and `SPOOLER_NO_TLS`, then set `SPOOLER_API_KEY`.

## Configuration

Connection, profile, logging, spool and queue flags accept `SPOOLER_`
environment variables. Flags override environment variables; other options are
per invocation.

Profiles store default settings in `~/.config/spooler/config.yaml` by default, `--config` overrides that.

A local testing from the [Usage](#usage) above can be configured like this:
```yaml
# config.yaml
profiles:
  local:
    host: 127.0.0.1:8080
    no-tls: true
    api-key: local
    spool: default
```

A flag overrides the profile, and the profile overrides an environment
variable, for every value a profile holds.
