# OpsRelay

OpsRelay is an authorized remote administration MVP. An administrator or AI-agent runs locally, uses the admin CLI, and sends audited jobs through the server to an explicitly enrolled client agent. The client machine does not run AI logic.

```text
AI-agent / administrator
        ↓
opsrelay CLI
        ↓
server API + WebSocket relay
        ↓
opsrelay-agent
        ↓
PowerShell/CMD or sh/bash
```

## Stack

```text
Server:       Node.js + TypeScript + Fastify
Admin CLI:    Node.js + TypeScript
Client Agent: Go
Protocol:     shared TypeScript/Zod contracts
Database:     PostgreSQL for deployed runtime; in-memory store remains available for local tests
Transport:    REST for admin/enrollment, WebSocket for agent jobs
```

## Quick Start

```bash
pnpm install
$env:OPSRELAY_ADMIN_TOKEN="dev-admin-token-change-me"
pnpm dev:server
```

In another terminal:

```bash
pnpm dev:cli -- enroll-token create --name local-agent --json
```

Enroll and run the agent after building it on a machine with Go installed:

```bash
cd agent
go run ./cmd/opsrelay-agent enroll --server http://127.0.0.1:3030 --token <token> --name local-agent --config ./opsrelay-agent.json
go run ./cmd/opsrelay-agent run --config ./opsrelay-agent.json
```

Then use the CLI:

```bash
pnpm dev:cli -- devices list --json
pnpm dev:cli -- session start <device-id> --name diagnostics --json
pnpm dev:cli -- sh <device-id> "uname -a" --json --save --session <session-id>
pnpm dev:cli -- ps <device-id> "Get-Service Spooler" --json --save --session <session-id>
pnpm dev:cli -- job status <job-id> --json
pnpm dev:cli -- job result <job-id> --json --save
pnpm dev:cli -- reboot <device-id> --confirm --reason "operator approved reboot" --json
```

Saved artifacts are written under `.sessions/<session-id>/`.

## Checks

```bash
pnpm typecheck
pnpm test
pnpm build

cd agent
go test ./...
go vet ./...
```

## Database

Local PostgreSQL can be started with:

```bash
docker compose up -d postgres
```

The SQL schema is in `apps/server/migrations/0001_init.sql`. If `OPSRELAY_DATABASE_URL` is set, the server uses PostgreSQL. Without that environment variable it falls back to an in-memory store for local tests.

## Safety Boundaries

OpsRelay is for authorized devices only. It implements enrollment tokens, device identity, admin API token auth, audit events, command timeouts, output limits and explicit reboot confirmation. It does not implement stealth install, EDR/AV bypass, credential dumping, keylogging, remote desktop or an interactive terminal in the MVP.

## What Exists Now

```text
apps/server       server API and WebSocket relay
apps/admin-cli    administrator CLI
agent             Go client agent for Windows/Linux
packages/protocol shared protocol contracts
```

The intended connection order is:

```text
Admin CLI -> Server -> Client Agent -> Local shell
```

The administrator never connects directly to the client machine. The client agent only connects outbound to the server.

## Server Deployment

Create `.env` from `.env.example` and set a long random `OPSRELAY_ADMIN_TOKEN`.

```bash
cp .env.example .env
docker compose -f docker-compose.deploy.yml --env-file .env up -d --build opsrelay-server
```

On Windows PowerShell, use `Copy-Item .env.example .env` instead of `cp`.

For real remote clients, expose the server through HTTPS with a reverse proxy such as Caddy, Nginx or a cloud load balancer:

```text
https://opsrelay.example.com -> http://127.0.0.1:3030
```

Health check:

```bash
curl https://opsrelay.example.com/health
```

For a deployed server, set `OPSRELAY_DATABASE_URL` so devices, sessions, jobs, results and audit events survive process restarts.

## Administrator Setup

On the administrator machine:

```bash
pnpm install
$env:OPSRELAY_SERVER_URL="https://opsrelay.example.com"
$env:OPSRELAY_ADMIN_TOKEN="<server-admin-token>"
pnpm dev:cli -- devices list --json
```

For this local checkout, admin CLI settings can be stored in `.env`:

```text
OPSRELAY_SERVER_URL=http://109.120.140.52:3030
OPSRELAY_ADMIN_TOKEN=<server-admin-token>
OPSRELAY_SESSION_DIR=.sessions
```

Load it in PowerShell before running CLI commands:

```powershell
Get-Content .env | ForEach-Object {
  if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
    [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim(), 'Process')
  }
}

node apps/admin-cli/dist/main.js devices list --json
```

Create a one-time enrollment token for a client:

```bash
pnpm operator token client-pc-01 --json
```

Run `pnpm build` first when using `pnpm operator`. The existing `enroll-token create --name` command remains available. The token appears once in the command output; pass it privately to the device owner.

## Two-machine operator mode

The operator CLI can run on the server host. This uses only the server and client machines while preserving the authenticated API, policy checks and audit trail. From the repository on the server host, with `OPSRELAY_ADMIN_TOKEN` set to the server's token:

```bash
pnpm build
pnpm operator token client-pc-01 --json
pnpm operator devices list --json
pnpm operator ps <device-id> "Get-Service Spooler" --json
```

For the Docker deployment, the CLI is included in the server image and points to its local API:

```bash
docker compose -f docker-compose.deploy.yml exec opsrelay-server node apps/admin-cli/dist/main.js token client-pc-01 --json
docker compose -f docker-compose.deploy.yml exec opsrelay-server node apps/admin-cli/dist/main.js devices list --json
docker compose -f docker-compose.deploy.yml exec opsrelay-server node apps/admin-cli/dist/main.js ps <device-id> "Get-Service Spooler" --json
```

The original three-machine mode remains available by running the same CLI on a separate administrator PC. No inbound connection to the client is required.

Give the client owner only:

```text
server URL
one-time enrollment token
agent exe
```

## Windows Client Agent

Build the Windows exe on a machine with Go installed:

```powershell
cd agent
.\scripts\build-windows.ps1
```

This creates:

```text
agent\dist\opsrelay-agent-windows-amd64.exe
```

This is the user-facing Windows GUI build. The client area uses a dark blue palette with mint accents and equally wide sections for connection status, device settings and controls. Actions and inline error messages use the same palette. The Russian-language window shows:

```text
Server URL
Enrollment token
Device name
Device ID
Color-coded connection status and last heartbeat time
Server check
```

First launch on a new client:

```text
1. Enter the server URL.
2. Paste the one-time enrollment token.
3. Confirm or edit the device name.
4. Click "Зарегистрировать и подключить".
5. Keep the window open while support is connected.
```

After enrollment, the agent saves its local config under:

```text
C:\ProgramData\OpsRelayAgent\config.json
```

Before enrollment, the GUI also remembers the last entered server URL and device name under:

```text
C:\ProgramData\OpsRelayAgent\prefs.json
```

The enrollment token is not saved because it is a one-time secret.

On later launches, the same exe finds the saved config, shows the Device ID and asks "Подключиться сейчас?" in the status area. The owner starts the connection with "Подключить сохранённое". The administrator sees the device as `online` only while connected.

The colored dot and status text distinguish connected (green), connecting/retrying (amber), disconnected (gray) and error (red). While connected, the activity line refreshes after each heartbeat. The `Server check` field and button show whether `/health` is reachable before enrollment or reconnect. The window uses the OpsRelay icon in the executable and taskbar.

For technical debugging, build a console variant:

```powershell
cd agent
.\scripts\build-windows.ps1 -Console -Output dist\opsrelay-agent-windows-amd64-console.exe
```

The console variant still supports manual enrollment:

```powershell
.\opsrelay-agent-windows-amd64-console.exe enroll `
  --server https://opsrelay.example.com `
  --token <one-time-enrollment-token> `
  --name "client-pc-01" `
  --config "$env:ProgramData\OpsRelayAgent\config.json"
```

And manual foreground run:

```powershell
.\opsrelay-agent-windows-amd64-console.exe run --config "$env:ProgramData\OpsRelayAgent\config.json"
```

The agent is visible and explicit. A Windows Service installer is not implemented yet; the packaging notes are in `agent/packaging/windows/README.md`.

## Admin Commands

```bash
pnpm dev:cli -- devices list --json
pnpm dev:cli -- session start <device-id> --name diagnostics --json
pnpm dev:cli -- ps <device-id> "Get-Service Spooler" --json --save --session <session-id>
pnpm dev:cli -- cmd <device-id> "whoami" --json --save --session <session-id>
pnpm dev:cli -- sh <device-id> "uname -a" --json --save --session <session-id>
pnpm dev:cli -- job status <job-id> --json
pnpm dev:cli -- job result <job-id> --json --save
pnpm dev:cli -- reboot <device-id> --confirm --reason "operator approved reboot" --json
```

Detailed deployment notes are in `docs/18-deployment-runbook.md`.

The runbook includes the full VPS flow used during testing: PostgreSQL installation, schema migration, systemd service setup, admin CLI environment variables, Windows client enrollment, and smoke commands for PowerShell/CMD route validation.
