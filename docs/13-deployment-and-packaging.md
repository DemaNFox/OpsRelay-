# Deployment and Packaging

## MVP Environments

```text
local-dev
staging
production, later
```

## Local Development

Recommended local stack:

```text
Docker Compose:
  PostgreSQL
  optional Redis/NATS later
  server

Host:
  admin-cli
  test agent
```

Planned command:

```bash
docker compose up -d postgres
pnpm install
pnpm --filter @remote-admin/server db:migrate
pnpm --filter @remote-admin/server dev
```

## Server Deployment

Recommended production-like layout:

```text
VPS or cloud VM
Docker container for server
PostgreSQL managed or containerized
Caddy/Nginx reverse proxy
TLS certificate
443/tcp only
```

The server should be reachable by both administrator and client devices.

## Windows Agent Packaging

MVP options:

```text
remote-agent.exe install --server <url> --token <token> --name <name>
remote-agent.exe uninstall
```

Production options:

```text
MSI installer
signed executable
signed installer
service recovery configuration
```

Expected paths:

```text
C:\Program Files\RemoteAdminAgent\remote-agent.exe
C:\ProgramData\RemoteAdminAgent\config.yaml
C:\ProgramData\RemoteAdminAgent\logs\
```

## Linux Agent Packaging

MVP options:

```bash
sudo ./remote-agent install --server <url> --token <token> --name <name>
sudo systemctl enable --now remote-agent
```

Production options:

```text
.deb for Debian/Ubuntu
.rpm for RHEL/Fedora
signed packages/repository
```

Expected paths:

```text
/usr/bin/remote-agent
/etc/remote-agent/config.yaml
/var/lib/remote-agent/
/var/log/remote-agent/
/etc/systemd/system/remote-agent.service
```

## Configuration

Server URL and device identity must be stored in config files readable only by the service account/root/Admin-equivalent.

## Updates

Auto-update is out of MVP scope. Architecture should keep room for:

```text
- signed update manifest;
- version checks;
- staged rollout;
- rollback;
- update audit events.
```

## Ports

For MVP, use outbound connections only from clients:

```text
443/tcp to server
```

No inbound ports should be required on client devices.
