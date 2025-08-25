# SSH Notifier (Go)

This agent streams SSH login activity from journald or system log files and posts notifications to Slack.

- Realtime daemon (default)
- Optional batch summary via systemd timer
- Rate limiting and basic deduplication (coming soon)

## Install

Option A: one-line install (latest stable release)

```bash
curl -fsSL https://raw.githubusercontent.com/raf181/SSH-Noty/main/install.sh | sudo bash -s -- --webhook="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

Option B: one-line install (Nightly prerelease)

```bash
curl -fsSL https://raw.githubusercontent.com/raf181/SSH-Noty/main/install.sh | sudo bash -s -- --webhook="https://hooks.slack.com/services/YOUR/WEBHOOK/URL" --tag=nightly
```

Option C: non-root install under a custom prefix (no systemd)

```bash
curl -fsSL https://raw.githubusercontent.com/raf181/SSH-Noty/main/install.sh | bash -s -- --prefix="$HOME/.local/ssh-noti" --no-systemd --tag=nightly
~/.local/ssh-noti/ssh-noti --test --config=~/.local/ssh-noti/config.json
```

You can also build locally and run the local installer:

```bash
go build -o ssh-noti ./cmd/ssh-noti
sudo ./install-ssh-noti.sh
```

Then edit `/opt/ssh-noti/config.json` and set `slack_webhook` (if not provided to the installer).

## Configuration

`/opt/ssh-noti/config.json`

- slack_webhook: Slack Incoming Webhook URL
- sources.prefer: auto | journald | file
- sources.file_paths: override text log locations
- sources.systemd_units: sshd.service, ssh.service
- telemetry.log_level: INFO | DEBUG | WARN | ERROR

## Systemd

- `ssh-noti.service` runs the realtime daemon
- `ssh-noti-summary.timer` triggers batch summary hourly (service stub provided)

## Troubleshooting

- Ensure the `sshnoti` user is in `adm` or `systemd-journal` group to read logs.
- Check logs:

```bash
journalctl -u ssh-noti -e
```

Releases

- Stable releases: [github.com/raf181/SSH-Noty/releases](https://github.com/raf181/SSH-Noty/releases)
- Nightly prerelease: tag "nightly" (updated by CI)
