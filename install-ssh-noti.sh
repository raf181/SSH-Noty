#!/usr/bin/env bash
set -euo pipefail

if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
  echo "Please run as root" >&2
  exit 1
fi

BIN_NAME="ssh-noti"
INSTALL_DIR="/opt/ssh-noti"
STATE_DIR="$INSTALL_DIR/state"
SERVICE_NAME="ssh-noti.service"
TIMER_NAME="ssh-noti-summary.timer"

echo "Installing $BIN_NAME to $INSTALL_DIR"
mkdir -p "$INSTALL_DIR" "$STATE_DIR"

# Prefer local binary if present
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
if [[ -x "$SCRIPT_DIR/$BIN_NAME" ]]; then
  install -m 0755 "$SCRIPT_DIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
else
  echo "Local binary not found at $SCRIPT_DIR/$BIN_NAME. Attempting to build..."
  if command -v go >/dev/null 2>&1; then
    (cd "$SCRIPT_DIR" && go build -o "$INSTALL_DIR/$BIN_NAME" ./cmd/ssh-noti)
  else
    echo "Go not installed and no binary available. Please build the binary first." >&2
    exit 1
  fi
fi

# Config
if [[ ! -f "$INSTALL_DIR/config.json" ]]; then
  if [[ -f "$SCRIPT_DIR/config.example.json" ]]; then
    install -m 0600 "$SCRIPT_DIR/config.example.json" "$INSTALL_DIR/config.json"
  else
    cat > "$INSTALL_DIR/config.json" <<'JSON'
{
  "slack_webhook": "",
  "mode": "realtime",
  "sources": { "prefer": "auto", "file_paths": ["/var/log/auth.log", "/var/log/secure"], "systemd_units": ["sshd.service", "ssh.service"] },
  "rules": { "notify_success": true, "notify_failure": true, "notify_invalid_user": true, "notify_root_login": true, "exclude_users": [], "exclude_ips": [], "include_ips": [] },
  "rate_limit": { "window_seconds": 60, "max_events_per_window": 20, "dedup_window_seconds": 30 },
  "batch": { "window_seconds": 3600, "min_failed_threshold": 5 },
  "geoip": { "enabled": false, "db_path": "/usr/share/GeoIP/GeoLite2-City.mmdb" },
  "formatting": { "concise": false, "show_key_fingerprint": true, "show_hostname": true },
  "telemetry": { "log_level": "INFO", "log_file": "/var/log/ssh-noti.log" }
}
JSON
    chmod 0600 "$INSTALL_DIR/config.json"
  fi
fi

# System user
if ! id -u sshnoti >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin sshnoti || true
fi

# Group access for logs
for g in adm systemd-journal; do
  if getent group "$g" >/dev/null 2>&1; then
    usermod -a -G "$g" sshnoti || true
  fi
done

chown -R root:root "$INSTALL_DIR"
chmod 0755 "$INSTALL_DIR"
chmod 0700 "$STATE_DIR"
chmod 0755 "$INSTALL_DIR/$BIN_NAME"
chmod 0600 "$INSTALL_DIR/config.json"

# Systemd units
UNIT_DIR="/etc/systemd/system"
install -m 0644 "$SCRIPT_DIR/systemd/ssh-noti.service" "$UNIT_DIR/ssh-noti.service"
install -m 0644 "$SCRIPT_DIR/systemd/ssh-noti-summary.service" "$UNIT_DIR/ssh-noti-summary.service"
install -m 0644 "$SCRIPT_DIR/systemd/ssh-noti-summary.timer" "$UNIT_DIR/ssh-noti-summary.timer"

systemctl daemon-reload
systemctl enable --now ssh-noti.service
systemctl enable --now ssh-noti-summary.timer || true

echo "Installation complete. Edit $INSTALL_DIR/config.json to set your Slack webhook."
echo "Test with: $INSTALL_DIR/$BIN_NAME --test --config=$INSTALL_DIR/config.json"
