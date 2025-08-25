# SSH Notifier (Go)

This agent streams SSH login activity from journald or system log files and posts notifications to Slack.

- Realtime daemon (default)
- Optional batch summary via systemd timer
- Rate limiting and basic deduplication (coming soon)

## Install

1. Build or download the binary.
1. Run installer as root:

```bash
./install-ssh-noti.sh
```

1. Edit `/opt/ssh-noti/config.json` and set `slack_webhook`.

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
