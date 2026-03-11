# sshmail Deployment

## VPS

- **Provider**: BinaryLane (Brisbane region)
- **Host**: shake-charlie.bnr.la
- **Public IP**: 43.229.61.163
- **User**: root

## Ports

| Port | Service |
|------|---------|
| 22   | sshmail (SSH — TUI + API) |
| 2200 | Admin SSH (sshd) — use this for server access |

## SSH Access

```bash
# Admin access
ssh -p 2200 root@43.229.61.163

# sshmail
ssh sshmail.dev
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
Environment=HUB_PORT=22
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
ssh sshmail.dev

# API (non-interactive)
ssh sshmail.dev inbox

# Git clone
git clone sshmail.dev:username
```

## Port Migration (2026-03-11)

Moved sshmail from port 2233 to port 22 so users can connect without specifying a port.

### Steps taken

1. **Removed port 22 from sshd** — edited `/etc/ssh/sshd_config`, kept only `Port 2200`
2. **Restarted sshd** — `systemctl restart sshd` (verified admin access still works on 2200)
3. **Updated sshmail service** — changed `HUB_PORT=2233` to `HUB_PORT=22` in systemd unit
4. **Restarted sshmail** — `systemctl daemon-reload && systemctl restart sshmail`
5. **Killed tarot tunnel** — port 22 was previously used for a reverse SSH tunnel to the tarot app

### Rollback

If sshmail needs to move off port 22:
```bash
ssh -p 2200 root@43.229.61.163
# Re-add Port 22 to /etc/ssh/sshd_config
# Change HUB_PORT back to 2233
systemctl restart sshd
systemctl daemon-reload && systemctl restart sshmail
```
