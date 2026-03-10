#!/usr/bin/env bash

set -euo pipefail

ROOT="/home/ubuntu/Dev/sshmail"
SYSTEMD_DIR="${HOME}/.config/systemd/user"

mkdir -p "$SYSTEMD_DIR"
chmod +x "${ROOT}/scripts/sshmail-autopilot.sh"

cat >"${SYSTEMD_DIR}/sshmail-autopilot.service" <<EOF
[Unit]
Description=sshmail autopilot poller

[Service]
Type=oneshot
WorkingDirectory=${ROOT}
ExecStart=${ROOT}/scripts/sshmail-autopilot.sh
EOF

cat >"${SYSTEMD_DIR}/sshmail-autopilot.timer" <<'EOF'
[Unit]
Description=Run sshmail autopilot every 10 minutes

[Timer]
OnBootSec=2min
OnUnitActiveSec=10min
Persistent=true

[Install]
WantedBy=timers.target
EOF

systemctl --user daemon-reload
systemctl --user enable --now sshmail-autopilot.timer
systemctl --user list-timers sshmail-autopilot.timer --no-pager
