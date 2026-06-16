# Admin CLI

## Purpose

`remote-admin` is the primary interface for administrators and AI-agents.

It must be stable, scriptable, machine-readable and safe by default.

## Design Goals

```text
- friendly for humans;
- reliable for AI-agents;
- JSON output for automation;
- stable exit codes;
- local session artifacts;
- no hidden direct access to client devices;
- all operations go through server API.
```

## Recommended Stack

```text
Node.js + TypeScript
Commander.js, Clipanion, or oclif
Zod/TypeBox validation
pino-pretty for local debug logs
```

## Command Groups

```text
remote-admin auth
remote-admin devices
remote-admin device
remote-admin session
remote-admin ps
remote-admin cmd
remote-admin sh
remote-admin job
remote-admin reboot
remote-admin collect
```

## Required Commands for MVP

```bash
remote-admin devices list [--json]
remote-admin device info <device-id> [--json]
remote-admin session start <device-id> --name <name> [--json]
remote-admin ps <device-id> <command> [--session <id>] [--json] [--save]
remote-admin cmd <device-id> <command> [--session <id>] [--json] [--save]
remote-admin sh <device-id> <command> [--session <id>] [--json] [--save]
remote-admin job status <job-id> [--json]
remote-admin job result <job-id> [--json] [--save]
remote-admin reboot <device-id> --confirm [--reason <text>] [--json]
```

## Machine-Readable Output

Every command used by AI-agent must support `--json`.

Success example:

```json
{
  "ok": true,
  "data": {
    "job_id": "job_123",
    "device_id": "client-001",
    "exit_code": 0,
    "stdout": "...",
    "stderr": "",
    "duration_ms": 532
  }
}
```

Failure example:

```json
{
  "ok": false,
  "error": {
    "code": "COMMAND_TIMED_OUT",
    "message": "Command timed out after 30 seconds",
    "retryable": false
  }
}
```

## Stable Exit Codes

```text
0   success
1   generic error
2   invalid arguments
3   authentication error
4   authorization error
5   device offline
6   job failed
7   command timed out
8   confirmation required
9   protocol/server incompatible
```

## Local Session Artifacts

When `--save` or a session is active, CLI must save artifacts locally:

```text
.sessions/
  <session-id>/
    session.json
    transcript.md
    commands/
      0001.command.ps1
      0002.command.sh
    results/
      0001.stdout.txt
      0001.stderr.txt
      0001.meta.json
    files/
    reports/
```

AI-agent should read these files locally instead of storing reasoning on the client device.

## CLI Config

Recommended config path:

```text
Windows: %APPDATA%\RemoteAdmin\config.yaml
Linux:   ~/.config/remote-admin/config.yaml
```

Example:

```yaml
server_url: "https://rmm.example.com"
admin_token_source: "env:REMOTE_ADMIN_TOKEN"
default_session_dir: ".sessions"
request_timeout_sec: 60
```

## Safety Rules

CLI must not provide any command that bypasses server policy. Destructive commands must require explicit flags such as `--confirm` and should create audit events.
