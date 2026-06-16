# System Architecture

## High-Level Architecture

```text
[Administrator PC]
  Codex / Claude Code / Human
  remote-admin CLI
  local session artifacts
        │
        │ HTTPS/WebSocket/gRPC
        ▼
[Server]
  API
  auth
  devices registry
  sessions
  jobs
  audit log
  agent gateway
        │
        │ persistent secure connection
        ▼
[Client Device]
  client-agent as Windows Service/systemd daemon
  shell executor
  platform adapters
  local logs
```

## Components

### Administrator PC

Runs:

```text
- AI-agent, for example Codex or Claude Code;
- remote-admin CLI;
- local session workspace;
- optional web browser for future UI.
```

The administrator PC is the only place where AI reasoning and task context should live.

### Server

The server is a control plane and relay. It does not perform diagnostics itself. It:

```text
- authenticates administrators;
- registers devices;
- accepts persistent client-agent connections;
- routes jobs to online devices;
- stores job results;
- stores sessions and audit events;
- enforces high-level policies.
```

### Client Agent

The client-agent is an execution component installed on the customer device.

It:

```text
- starts automatically;
- connects outbound to the server;
- sends heartbeat and capabilities;
- receives jobs;
- executes commands with timeout and output limits;
- returns structured results;
- reconnects after network failures or reboot.
```

It must not perform AI reasoning or hide its presence.

## Network Model

Both administrator and client connect outbound to the server:

```text
Administrator PC ───► Server ◄─── Client Device
```

This avoids direct inbound connectivity to client devices and works behind NAT/CG-NAT/firewalls in typical environments.

## Trust Boundaries

```text
Administrator PC → trusted operator environment
Server           → trusted control plane
Client Agent     → trusted execution component on authorized device
Client OS        → execution environment, potentially unstable/untrusted state
Network          → untrusted, must use TLS
```

## Data Flow: Shell Job

```text
1. CLI sends job request to server.
2. Server validates admin token and device permissions.
3. Server creates job record.
4. Server forwards job to connected client-agent.
5. Agent validates job format and local policy.
6. Agent executes command with timeout.
7. Agent sends result to server.
8. Server stores result and notifies CLI.
9. CLI writes local session artifacts.
10. AI-agent reads local artifacts and decides next step.
```

## Job-Based Execution

MVP should use job-based execution, not interactive shell. This improves:

```text
- auditability;
- reproducibility;
- timeout handling;
- result persistence;
- AI-agent parsing;
- reconnect behavior;
- approval flow.
```

Interactive shell can be added later as a separate feature.
