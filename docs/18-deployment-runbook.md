# Deployment Runbook

This runbook describes the MVP deployment flow. For deployed usage, set `OPSRELAY_DATABASE_URL` and apply `apps/server/migrations/0001_init.sql` so devices, sessions, jobs, results and audit events persist across restarts.

## Server

### Option A: systemd on Ubuntu VPS

This is the flow used for the first external route test.

1. Copy the repository to a VPS or VM.
2. Install PostgreSQL:

```bash
apt-get update
apt-get install -y postgresql postgresql-contrib
systemctl enable --now postgresql
```

3. Create database and role:

```bash
runuser -u postgres -- psql
```

```sql
CREATE ROLE opsrelay LOGIN PASSWORD '<long-random-db-password>';
CREATE DATABASE opsrelay OWNER opsrelay;
GRANT ALL PRIVILEGES ON DATABASE opsrelay TO opsrelay;
```

4. Install Node dependencies and build:

```bash
cd /opt/opsrelay
corepack enable
corepack prepare pnpm@10.33.2 --activate
pnpm install --frozen-lockfile
pnpm build
```

5. Apply the schema:

```bash
PGPASSWORD='<long-random-db-password>' \
  psql -h 127.0.0.1 -U opsrelay -d opsrelay \
  -v ON_ERROR_STOP=1 \
  -f apps/server/migrations/0001_init.sql
```

6. Create `/opt/opsrelay/.env`:

```bash
OPSRELAY_SERVER_HOST=0.0.0.0
OPSRELAY_SERVER_PORT=3030
OPSRELAY_ADMIN_TOKEN=<long-random-admin-token>
OPSRELAY_LOG_LEVEL=info
OPSRELAY_DATABASE_URL=postgres://opsrelay:<long-random-db-password>@127.0.0.1:5432/opsrelay
```

7. Create `/etc/systemd/system/opsrelay-server.service`:

```ini
[Unit]
Description=OpsRelay Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/opsrelay
EnvironmentFile=/opt/opsrelay/.env
ExecStart=/usr/bin/node apps/server/dist/main.js
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
```

8. Start the service:

```bash
systemctl daemon-reload
systemctl enable opsrelay-server
systemctl restart opsrelay-server
systemctl status opsrelay-server
```

9. Health check:

```bash
curl http://127.0.0.1:3030/health
curl http://server-ip:3030/health
```

10. Confirm persistence after a restart:

```bash
systemctl restart opsrelay-server
curl http://127.0.0.1:3030/health
psql "$OPSRELAY_DATABASE_URL" -tAc "select count(*) from devices; select count(*) from jobs; select count(*) from job_results; select count(*) from audit_events;"
```

### Option B: Docker Compose

Create `.env` from `.env.example`, set long random secrets, set `OPSRELAY_DATABASE_URL`, and start:

```bash
docker compose -f docker-compose.deploy.yml --env-file .env up -d --build opsrelay-server
```

Health check:

```bash
curl https://opsrelay.example.com/health
```

For temporary local/LAN testing without TLS:

```bash
curl http://server-ip:3030/health
```

For real remote clients, put a reverse proxy with TLS in front of the server:

```text
https://opsrelay.example.com -> http://127.0.0.1:3030
```

## Admin Machine

The administrator uses the CLI only. The CLI talks to the server API and never connects directly to client machines.

```bash
pnpm install
export OPSRELAY_SERVER_URL=https://opsrelay.example.com
export OPSRELAY_ADMIN_TOKEN=<same-admin-token-as-server>
pnpm dev:cli -- devices list --json
```

Create an enrollment token with one command after building the workspace:

```bash
pnpm operator token client-pc-01 --json
```

Give only the one-time enrollment token and server URL to the device owner.

The administrator can also work directly on the server host, so only the server and client machines are involved. With the server's `OPSRELAY_ADMIN_TOKEN` in the shell environment, run `pnpm operator devices list --json` or `pnpm operator ps <device-id> "Get-Service Spooler" --json`. The CLI still calls the authenticated local server API and jobs are audited. For Docker deployment, run the bundled CLI inside the server container:

```bash
docker compose -f docker-compose.deploy.yml exec opsrelay-server node apps/admin-cli/dist/main.js token client-pc-01 --json
docker compose -f docker-compose.deploy.yml exec opsrelay-server node apps/admin-cli/dist/main.js devices list --json
docker compose -f docker-compose.deploy.yml exec opsrelay-server node apps/admin-cli/dist/main.js ps <device-id> "Get-Service Spooler" --json
```

## Windows Client

Build the agent exe on a machine with Go:

```powershell
cd agent
.\scripts\build-windows.ps1
```

Copy `agent\dist\opsrelay-agent-windows-amd64.exe` to the Windows client.

Normal user flow:

```text
1. Double-click opsrelay-agent-windows-amd64.exe.
2. Enter server URL.
3. Paste the one-time enrollment token.
4. Confirm the device name.
5. Click "Зарегистрировать и подключить".
6. Keep the window open while the administrator works.
```

The window shows Device ID, a colored connection status and server health status. On later launches, the owner is asked whether to connect using the saved config; declining leaves the manual "Подключить сохранённое" button available.

Technical console flow:

```powershell
.\scripts\build-windows.ps1 -Console -Output dist\opsrelay-agent-windows-amd64-console.exe
```

```powershell
.\opsrelay-agent-windows-amd64-console.exe enroll `
  --server https://opsrelay.example.com `
  --token <one-time-enrollment-token> `
  --name "client-pc-01" `
  --config "$env:ProgramData\OpsRelayAgent\config.json"
```

Run foreground:

```powershell
.\opsrelay-agent-windows-amd64-console.exe run --config "$env:ProgramData\OpsRelayAgent\config.json"
```

The GUI foreground mode is intentional for MVP validation. A production Windows Service installer is not implemented yet.

## Linux Client

Build:

```bash
cd agent
sh scripts/build-linux.sh
```

Enroll:

```bash
sudo ./dist/opsrelay-agent-linux-amd64 enroll \
  --server https://opsrelay.example.com \
  --token <one-time-enrollment-token> \
  --name client-linux-01 \
  --config /etc/opsrelay-agent/config.json
```

Run:

```bash
sudo ./dist/opsrelay-agent-linux-amd64 run --config /etc/opsrelay-agent/config.json
```

## Basic Operator Flow

Set admin environment:

```bash
export OPSRELAY_SERVER_URL=http://server-ip:3030
export OPSRELAY_ADMIN_TOKEN=<server-admin-token>
```

```bash
pnpm dev:cli -- devices list --json
pnpm dev:cli -- session start <device-id> --name diagnostics --json
pnpm dev:cli -- ps <device-id> "Get-Service Spooler" --json --save --session <session-id>
pnpm dev:cli -- sh <device-id> "uname -a" --json --save --session <session-id>
pnpm dev:cli -- job status <job-id> --json
pnpm dev:cli -- job result <job-id> --json --save
```

Windows route smoke used during external testing:

```powershell
$env:OPSRELAY_SERVER_URL="http://server-ip:3030"
$env:OPSRELAY_ADMIN_TOKEN="<server-admin-token>"

node apps/admin-cli/dist/main.js devices list --json
node apps/admin-cli/dist/main.js session start <device-id> --name route-smoke --json
node apps/admin-cli/dist/main.js ps <device-id> "Write-Output route-print-ok" --session <session-id> --timeout 10 --json
node apps/admin-cli/dist/main.js ps <device-id> "Test-Connection 127.0.0.1 -Count 2" --session <session-id> --timeout 20 --json
node apps/admin-cli/dist/main.js cmd <device-id> "ping 127.0.0.1 -n 2" --session <session-id> --timeout 20 --json
node apps/admin-cli/dist/main.js cmd <device-id> "echo Hello from CMD through OpsRelay" --session <session-id> --timeout 10 --json
```

Expected result: every command returns a JSON envelope with `"ok": true`, job status `completed`, `exit_code: 0`, and stdout from the remote client shell.

Reboot requires explicit confirmation:

```bash
pnpm dev:cli -- reboot <device-id> --confirm --reason "operator approved reboot" --json
```

## Required Hardening Before Real Production

- Put the server behind HTTPS.
- Package the Windows agent as a visible Windows Service installer.
- Add signed binaries/installers.
- Add device revocation and token rotation UI/CLI.
