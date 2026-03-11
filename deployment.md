# sshmail Deployment

## VPS

- **Provider**: BinaryLane (Brisbane region)
- **Host**: shake-charlie.bnr.la
- **Public IP**: 43.229.61.163
- **User**: root

## Ports

| Port | Service |
|------|---------|
| 22   | Tarot app SSH tunnel |
| 2200 | Real SSH (sshd) — use this for admin access |
| 2222 | sshmail API (JSON over SSH, no PTY) |
| 2233 | sshmail TUI (SSH with PTY, Bubble Tea UI) |

## SSH Access

```bash
ssh -p 2200 root@43.229.61.163
```

## File Locations

| Path | Description |
|------|-------------|
| `/usr/local/bin/sshmail` | Hub binary |
| `/var/lib/sshmail/` | Data directory (db, keys, repos, files) |
| `/var/log/sshmail.log` | Log file (only used for manual runs) |
| `/etc/systemd/system/sshmail.service` | Systemd unit file |

## Systemd Service

The service is managed via systemd:

```bash
# Status
systemctl status sshmail

# Restart
systemctl restart sshmail

# Logs (journald)
journalctl -u sshmail -f

# Logs (manual runs only)
tail -f /var/log/sshmail.log
```

### Service config (`/etc/systemd/system/sshmail.service`)

```ini
[Unit]
Description=sshmail hub
After=network.target

[Service]
ExecStart=/usr/local/bin/sshmail
WorkingDirectory=/var/lib/sshmail
Environment=HUB_PORT=2233
Environment=BBS_DATA_DIR=/var/lib/sshmail
Environment=BBS_ADMIN_KEY=/root/.ssh/authorized_keys
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## Deploy

Build the binary locally, upload, and restart the service:

```bash
# 1. Build linux/amd64 binary
GOOS=linux GOARCH=amd64 go build -o hub-linux ./cmd/hub/

# 2. Upload to VPS
scp -P 2200 hub-linux root@43.229.61.163:/tmp/sshmail-hub-new

# 3. Swap binary and restart
ssh -p 2200 root@43.229.61.163 'cp /tmp/sshmail-hub-new /usr/local/bin/sshmail && chmod +x /usr/local/bin/sshmail && systemctl restart sshmail'

# 4. Verify
ssh -p 2200 root@43.229.61.163 'systemctl status sshmail --no-pager'
```

## Testing

```bash
# TUI (interactive)
ssh -p 2233 ssh.sshmail.dev

# API (non-interactive)
ssh -p 2233 ssh.sshmail.dev inbox
```
