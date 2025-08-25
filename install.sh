#!/usr/bin/env bash
set -euo pipefail

# Usage examples:
#   curl -fsSL https://raw.githubusercontent.com/raf181/SSH-Noty/main/install.sh | sudo bash -s -- --webhook="https://hooks.slack.com/services/..."

OWNER_REPO="raf181/SSH-Noty"
BIN_NAME="ssh-noti"
INSTALL_DIR="/opt/ssh-noti"
STATE_DIR="$INSTALL_DIR/state"

WEBHOOK=""
MODE="realtime"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --webhook=*) WEBHOOK="${1#*=}"; shift ;;
    --mode=*) MODE="${1#*=}"; shift ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
  echo "Please run as root" >&2
  exit 1
fi

command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 1; }

echo "Creating directories..."
mkdir -p "$INSTALL_DIR" "$STATE_DIR"

echo "Fetching latest release for $OWNER_REPO..."
# Use GitHub releases redirect to determine latest tag without jq
LATEST_URL=$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/${OWNER_REPO}/releases/latest)
LATEST_TAG=${LATEST_URL##*/}
ASSET_URL_AMD64="https://github.com/${OWNER_REPO}/releases/download/${LATEST_TAG}/ssh-noti_linux_amd64"
ASSET_URL_ARM64="https://github.com/${OWNER_REPO}/releases/download/${LATEST_TAG}/ssh-noti_linux_arm64"

ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) DL_URL="$ASSET_URL_AMD64" ;;
  aarch64|arm64) DL_URL="$ASSET_URL_ARM64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

if [[ -z "$DL_URL" || "$DL_URL" == "null" ]]; then
  echo "No binary found in the latest release for arch $ARCH" >&2
  exit 1
fi

TMP_BIN=$(mktemp)
echo "Downloading $DL_URL ..."
curl -fsSL "$DL_URL" -o "$TMP_BIN"
install -m 0755 "$TMP_BIN" "$INSTALL_DIR/$BIN_NAME"
rm -f "$TMP_BIN"

# Write config
if [[ ! -f "$INSTALL_DIR/config.json" ]]; then
  cat > "$INSTALL_DIR/config.json" <<JSON
{
  "slack_webhook": "${WEBHOOK}",
  "mode": "${MODE}",
  "sources": { "prefer": "auto", "file_paths": ["/var/log/auth.log", "/var/log/secure"], "systemd_units": ["sshd.service", "ssh.service"] },
  "rules": { "notify_success": true, "notify_failure": true, "notify_invalid_user": true, "notify_root_login": true, "exclude_users": [], "exclude_ips": [], "include_ips": [] },
  "rate_limit": { "window_seconds": 60, "max_events_per_window": 20, "dedup_window_seconds": 30 },
  "batch": { "window_seconds": 3600, "min_failed_threshold": 5 },
  "geoip": { "enabled": false, "db_path": "/usr/share/GeoIP/GeoLite2-City.mmdb" },
  "formatting": { "concise": false, "show_key_fingerprint": true, "show_hostname": true },
  "telemetry": { "log_level": "INFO", "log_file": "/var/log/ssh-noti.log" }
}
JSON
fi

chmod 0600 "$INSTALL_DIR/config.json"

# Create system user and perms
if ! id -u sshnoti >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin sshnoti || true
fi
for g in adm systemd-journal; do
  if getent group "$g" >/dev/null 2>&1; then
    usermod -a -G "$g" sshnoti || true
  fi
done

chown -R root:root "$INSTALL_DIR"
chmod 0755 "$INSTALL_DIR"
chmod 0700 "$STATE_DIR"

# Install systemd units (inline)
if command -v systemctl >/dev/null 2>&1; then
  UNIT_DIR="/etc/systemd/system"
  mkdir -p "$UNIT_DIR"

  cat > "$UNIT_DIR/ssh-noti.service" <<'UNIT'
[Unit]
Description=SSH Notifier (Go)
After=network-online.target

[Service]
Type=simple
User=sshnoti
Group=sshnoti
ExecStart=/opt/ssh-noti/ssh-noti --daemon --config=/opt/ssh-noti/config.json
Restart=always
RestartSec=5
AmbientCapabilities=
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
UNIT

  cat > "$UNIT_DIR/ssh-noti-summary.service" <<'UNIT'
[Unit]
Description=SSH Notifier Summary Run

[Service]
Type=oneshot
User=sshnoti
Group=sshnoti
ExecStart=/opt/ssh-noti/ssh-noti --batch --config=/opt/ssh-noti/config.json
UNIT

  cat > "$UNIT_DIR/ssh-noti-summary.timer" <<'UNIT'
[Unit]
Description=SSH Notifier Summary (hourly)

[Timer]
OnCalendar=hourly
Persistent=true

[Install]
WantedBy=timers.target
UNIT

  chmod 0644 "$UNIT_DIR/ssh-noti.service" "$UNIT_DIR/ssh-noti-summary.service" "$UNIT_DIR/ssh-noti-summary.timer"
  systemctl daemon-reload
  systemctl enable --now ssh-noti.service
  systemctl enable --now ssh-noti-summary.timer || true
  echo "Systemd service installed and started: ssh-noti.service"
else
  echo "systemctl not found; skipping service installation. You can run $INSTALL_DIR/$BIN_NAME --daemon manually or set up a cron/timer alternative." >&2
fi

echo "Installed ssh-noti to $INSTALL_DIR"
echo "Configure webhook in $INSTALL_DIR/config.json if not provided via --webhook"
echo "Test: $INSTALL_DIR/$BIN_NAME --test --config=$INSTALL_DIR/config.json"
