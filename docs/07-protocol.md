# Protocol

## Purpose

The protocol defines how server, CLI and client-agent exchange data. It must be independent of a specific AI-agent and independent of OS-specific details.

## Recommended Options

For MVP:

```text
JSON over WebSocket + JSON Schema validation
```

For later production hardening:

```text
gRPC streaming + Protobuf
```

The project should keep protocol contracts in `packages/protocol` and generate TypeScript/Go types where possible.

## Versioning

Every agent connection must declare:

```text
protocol_version
agent_version
os type/version
capabilities
```

Example:

```json
{
  "type": "hello",
  "protocol_version": "0.1.0",
  "agent_version": "0.1.0",
  "device_id": "dev_123",
  "os": {
    "type": "windows",
    "version": "11"
  },
  "capabilities": [
    "shell.powershell",
    "shell.cmd",
    "reboot"
  ]
}
```

## Core Entities

### Device

```json
{
  "id": "dev_123",
  "name": "office-pc-01",
  "status": "online",
  "os": {
    "type": "windows",
    "version": "11"
  },
  "capabilities": ["shell.powershell", "shell.cmd", "reboot"],
  "last_seen_at": "2026-06-16T10:00:00Z"
}
```

### Job Request

```json
{
  "job_id": "job_123",
  "session_id": "ses_123",
  "device_id": "dev_123",
  "kind": "shell.exec",
  "shell": "powershell",
  "command": "Get-Service Spooler",
  "timeout_sec": 30,
  "stdout_limit_bytes": 1048576,
  "stderr_limit_bytes": 1048576,
  "created_by": "admin_123"
}
```

### Job Result

```json
{
  "job_id": "job_123",
  "device_id": "dev_123",
  "status": "completed",
  "exit_code": 0,
  "stdout": "...",
  "stderr": "",
  "stdout_truncated": false,
  "stderr_truncated": false,
  "started_at": "2026-06-16T10:00:01Z",
  "finished_at": "2026-06-16T10:00:02Z",
  "duration_ms": 1000
}
```

### Heartbeat

```json
{
  "type": "heartbeat",
  "device_id": "dev_123",
  "agent_version": "0.1.0",
  "uptime_sec": 3600,
  "active_jobs": 0,
  "sent_at": "2026-06-16T10:00:00Z"
}
```

## Supported Shells

```text
Windows:
  powershell
  cmd

Linux:
  sh
  bash, optional if available
```

Server must validate that the requested shell is supported by device capabilities.

## Capabilities

Capabilities are feature flags reported by the agent:

```text
shell.powershell
shell.cmd
shell.sh
shell.bash
services.windows
services.systemd
logs.eventlog
logs.journald
files.read
files.write
reboot
```

## Job Kinds for MVP

```text
shell.exec
system.reboot
```

Future job kinds:

```text
file.get
file.put
service.status
service.restart
logs.query
package.list
package.install
```

## Error Format

Protocol-level errors:

```json
{
  "ok": false,
  "error": {
    "code": "UNSUPPORTED_CAPABILITY",
    "message": "Device does not support shell.powershell",
    "retryable": false
  }
}
```

## Compatibility Rule

Server must reject incompatible protocol versions with a clear error and should never silently misinterpret messages.
