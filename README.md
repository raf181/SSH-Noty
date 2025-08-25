# SSH-Noty (Go)

Lightweight Linux agent that detects SSH successful and failed logins in near real-time and posts notifications to Slack. Supports journald and plain-text log tailing, rate limiting, deduplication, and optional batch summaries.

Status: initial scaffold with realtime daemon, Slack test, and basic journald/file parsing.

## Build

- Requires Go 1.22+
- Build binary:

```bash
go build ./cmd/ssh-noti
```

Binary at `./ssh-noti`.

## Quick run

- Create a config at `/opt/ssh-noti/config.json` or local `./config.json` with your Slack webhook.
- Test mode:

```bash
./ssh-noti --test --config=./config.json
```

- Daemon (foreground):

```bash
./ssh-noti --daemon --config=./config.json
```

## Install as service

Option 1: one-line install (downloads latest release):

```bash
curl -fsSL https://raw.githubusercontent.com/raf181/SSH-Noty/main/install.sh | sudo bash -s -- --webhook="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

Option 2: local install script (from repo): see `install-ssh-noti.sh`.

More details in `docs/SSH-NOTI.md`.
