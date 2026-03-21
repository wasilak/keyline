---
sidebar_label: Binary Installation
sidebar_position: 3
---

# Binary Installation

Install Keyline as a standalone binary on Linux, macOS, or Windows for bare-metal deployments.

## Overview

Keyline provides pre-compiled binaries for major platforms. This guide covers installation, configuration, and running Keyline as a system service.

## Download Binaries

### Latest Release

```bash
# Linux (amd64)
curl -LO https://github.com/wasilak/keyline/releases/latest/download/keyline-linux-amd64.tar.gz
tar -xzf keyline-linux-amd64.tar.gz
sudo mv keyline /usr/local/bin/

# Linux (arm64)
curl -LO https://github.com/wasilak/keyline/releases/latest/download/keyline-linux-arm64.tar.gz
tar -xzf keyline-linux-arm64.tar.gz
sudo mv keyline /usr/local/bin/

# macOS (Intel)
curl -LO https://github.com/wasilak/keyline/releases/latest/download/keyline-darwin-amd64.tar.gz
tar -xzf keyline-darwin-amd64.tar.gz
sudo mv keyline /usr/local/bin/

# macOS (Apple Silicon)
curl -LO https://github.com/wasilak/keyline/releases/latest/download/keyline-darwin-arm64.tar.gz
tar -xzf keyline-darwin-arm64.tar.gz
sudo mv keyline /usr/local/bin/

# Windows (amd64)
curl -LO https://github.com/wasilak/keyline/releases/latest/download/keyline-windows-amd64.zip
Expand-Archive keyline-windows-amd64.zip
Move-Item keyline-windows-amd64\keyline.exe C:\Windows\System32\
```

### Verify Installation

```bash
# Check version
keyline --version

# Expected output:
# keyline version 1.0.0
```

## Configuration

### Create Configuration Directory

```bash
sudo mkdir -p /etc/keyline
sudo mkdir -p /var/log/keyline
```

### Create Configuration File

```bash
sudo cat > /etc/keyline/config.yaml << 'EOF'
server:
  port: 9000
  mode: forward_auth

oidc:
  enabled: true
  issuer_url: ${OIDC_ISSUER_URL}
  client_id: ${OIDC_CLIENT_ID}
  client_secret: ${OIDC_CLIENT_SECRET}
  redirect_url: https://auth.example.com/auth/callback

session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: .example.com

cache:
  backend: redis
  redis_url: redis://localhost:6379
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}

user_management:
  enabled: true
  password_length: 32
  credential_ttl: 1h

role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser

default_es_roles:
  - viewer

elasticsearch:
  admin_user: keyline_admin
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
EOF
```

### Create Environment File

```bash
sudo cat > /etc/keyline/keyline.env << 'EOF'
# Session secret (min 32 bytes)
SESSION_SECRET=your-session-secret-here

# Cache encryption key (exactly 32 bytes)
CACHE_ENCRYPTION_KEY=your-encryption-key-here

# Elasticsearch admin password
ES_ADMIN_PASSWORD=your-es-admin-password

# OIDC client secret
OIDC_CLIENT_SECRET=your-oidc-client-secret

# Redis URL (optional)
REDIS_URL=redis://localhost:6379
EOF
```

### Secure Permissions

```bash
sudo chmod 600 /etc/keyline/config.yaml
sudo chmod 600 /etc/keyline/keyline.env
sudo chown root:root /etc/keyline/config.yaml
sudo chown root:root /etc/keyline/keyline.env
```

### Validate Configuration

```bash
# Source environment
set -a
source /etc/keyline/keyline.env
set +a

# Validate configuration
sudo keyline --validate-config --config /etc/keyline/config.yaml
```

## Run as System Service

### systemd (Linux)

#### Create Service Unit

```bash
sudo cat > /etc/systemd/system/keyline.service << 'EOF'
[Unit]
Description=Keyline Authentication Proxy
Documentation=https://github.com/wasilak/keyline
After=network.target redis.service
Wants=redis.service

[Service]
Type=simple
User=keyline
Group=keyline
EnvironmentFile=/etc/keyline/keyline.env
ExecStart=/usr/local/bin/keyline --config /etc/keyline/config.yaml
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/keyline

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=keyline

[Install]
WantedBy=multi-user.target
EOF
```

#### Create User and Directories

```bash
# Create keyline user
sudo useradd --system --no-create-home --shell /usr/sbin/nologin keyline

# Create directories
sudo mkdir -p /var/log/keyline
sudo chown keyline:keyline /var/log/keyline

# Set permissions
sudo chmod 755 /etc/keyline
```

#### Enable and Start Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service
sudo systemctl enable keyline

# Start service
sudo systemctl start keyline

# Check status
sudo systemctl status keyline
```

#### View Logs

```bash
# Journalctl logs
sudo journalctl -u keyline -f

# Last 100 lines
sudo journalctl -u keyline -n 100
```

### launchd (macOS)

#### Create Launch Daemon

```bash
sudo cat > /Library/LaunchDaemons/com.github.keyline.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.github.keyline</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/keyline</string>
        <string>--config</string>
        <string>/etc/keyline/config.yaml</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>SESSION_SECRET</key>
        <string>your-session-secret</string>
        <key>CACHE_ENCRYPTION_KEY</key>
        <string>your-encryption-key</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/keyline/keyline.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/keyline/keyline.err.log</string>
</dict>
</plist>
EOF
```

#### Load and Start

```bash
# Load daemon
sudo launchctl load /Library/LaunchDaemons/com.github.keyline.plist

# Start daemon
sudo launchctl start com.github.keyline

# Check status
sudo launchctl list | grep keyline
```

### Windows Service

#### Using NSSM (Non-Sucking Service Manager)

```powershell
# Download NSSM
curl -LO https://nssm.cc/release/nssm-2.24.zip
Expand-Archive nssm-2.24.zip
cd nssm-2.24\win64

# Install service
.\nssm.exe install keyline "C:\Program Files\keyline\keyline.exe" "--config C:\Program Files\keyline\config.yaml"

# Set environment variables
.\nssm.exe set keyline AppEnvironmentExtra "SESSION_SECRET=your-secret;CACHE_ENCRYPTION_KEY=your-key"

# Start service
.\nssm.exe start keyline

# Check status
.\nssm.exe status keyline
```

## Testing

### Test Health Endpoint

```bash
curl http://localhost:9000/healthz
```

### Test Authentication

```bash
# Without auth (should redirect)
curl -v http://localhost:9000/

# With Basic Auth
curl -v -u admin:password http://localhost:9000/_cluster/health
```

### Test Metrics

```bash
curl http://localhost:9000/_metrics
```

## Upgrading

### Manual Upgrade

```bash
# Stop service
sudo systemctl stop keyline

# Backup current binary
sudo cp /usr/local/bin/keyline /usr/local/bin/keyline.bak

# Download new version
curl -LO https://github.com/wasilak/keyline/releases/latest/download/keyline-linux-amd64.tar.gz
tar -xzf keyline-linux-amd64.tar.gz
sudo mv keyline /usr/local/bin/

# Validate new version
sudo keyline --version

# Start service
sudo systemctl start keyline

# Verify
sudo systemctl status keyline
```

### Rollback

```bash
# Stop service
sudo systemctl stop keyline

# Restore backup
sudo cp /usr/local/bin/keyline.bak /usr/local/bin/keyline

# Start service
sudo systemctl start keyline
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
sudo journalctl -u keyline -n 100

# Test configuration
sudo keyline --validate-config --config /etc/keyline/config.yaml

# Check permissions
ls -la /etc/keyline/
```

### Port Already in Use

```bash
# Check what's using port 9000
sudo lsof -i :9000

# Change port in config
sudo nano /etc/keyline/config.yaml
# server.port: 9001
```

### Can't Connect to Redis

```bash
# Test Redis connectivity
redis-cli ping

# Check Redis service
sudo systemctl status redis
```

## Next Steps

- **[Docker Deployment](./docker.md)** - Docker deployment guide
- **[Kubernetes Deployment](./kubernetes.md)** - K8s deployment
- **[Security Best Practices](./security-best-practices.md)** - Security guidelines
