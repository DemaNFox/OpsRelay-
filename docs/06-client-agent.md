# Client Agent

## Purpose

The client-agent is a Go service/daemon installed on an authorized client device. It only executes explicit jobs received from the server and returns structured results.

It must not perform AI reasoning.

## Recommended Stack

```text
Language: Go
Transport: WebSocket or gRPC streaming over TLS
Config: YAML/TOML/JSON
Logging: slog or zap/zerolog
Packaging: MSI/EXE for Windows, deb/rpm/tar for Linux
Service: Windows Service, systemd
```

## Core Responsibilities

```text
- enroll device;
- store device identity/config;
- connect outbound to server;
- send heartbeat;
- report capabilities;
- receive jobs;
- execute jobs with timeout/cancellation;
- limit output size;
- send results;
- reconnect after failures;
- start after reboot;
- write local logs.
```

## Architecture

```text
agent/
  cmd/remote-agent/main.go
  internal/core/
    agent.go
    heartbeat.go
    jobs.go
    capabilities.go
  internal/transport/
    client.go
    websocket.go
  internal/executor/
    shell.go
    process.go
  internal/platform/
    platform.go
    windows/
    linux/
  internal/config/
  internal/logging/
  internal/security/
  packaging/
    windows/
    linux/
```

## Platform Abstractions

The core must depend on interfaces, not OS-specific implementations.

```go
// Conceptual shape only.
type Platform interface {
    OSInfo(ctx context.Context) (OSInfo, error)
    Capabilities(ctx context.Context) ([]Capability, error)
    Reboot(ctx context.Context, reason string) error
}

type ShellExecutor interface {
    Run(ctx context.Context, request ShellRequest) (ShellResult, error)
}
```

## Windows Capabilities

```text
shell.cmd
shell.powershell
services.windows
logs.eventlog, later
reboot
files.read
files.write, later
```

## Linux/Ubuntu Capabilities

```text
shell.sh
services.systemd
logs.journald, later
packages.apt, later
reboot
files.read
files.write, later
```

## Installation Paths

### Windows

```text
Binary: C:\Program Files\RemoteAdminAgent\remote-agent.exe
Config: C:\ProgramData\RemoteAdminAgent\config.yaml
Logs:   C:\ProgramData\RemoteAdminAgent\logs\
Service name: RemoteAdminAgent
```

### Linux

```text
Binary: /usr/bin/remote-agent
Config: /etc/remote-agent/config.yaml
Data:   /var/lib/remote-agent/
Logs:   /var/log/remote-agent/
Service: remote-agent.service
```

## Enrollment

Enrollment should use a one-time token:

```bash
remote-agent enroll --server https://rmm.example.com --token <one-time-token> --name "office-pc-01"
```

After enrollment, the agent receives/stores:

```text
device_id
device_name
server_url
protocol_version
device secret/private key/certificate material
```

## Job Execution Rules

Every command execution must include:

```text
- timeout;
- cancellation;
- stdout limit;
- stderr limit;
- working directory policy;
- environment filtering;
- exit code capture;
- start/end timestamps;
- correlation IDs.
```

## Reconnect Strategy

Use exponential backoff with jitter:

```text
1s → 2s → 5s → 10s → 30s → 60s max
```

After reboot, agent reconnects and sends heartbeat with updated boot/session information.

## Non-Goals

The agent must not implement:

```text
- stealth mode;
- persistence beyond explicit service/daemon install;
- credential dumping;
- keylogging;
- EDR/AV bypass;
- unauthorized screen capture;
- hidden network tunneling outside the defined protocol.
```
