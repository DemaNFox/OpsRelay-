# MVP Scope

## MVP v0 Objective

Build the smallest reliable platform that can enroll devices, execute remote shell jobs, persist results and support AI-assisted debugging through CLI.

## In Scope

### Server

```text
- administrator API token auth;
- device enrollment tokens;
- device registry;
- agent WebSocket/gRPC gateway;
- device heartbeat;
- device capabilities;
- job creation;
- job dispatch;
- job result storage;
- debug sessions;
- audit log;
- basic PostgreSQL migrations.
```

### Admin CLI

```text
- devices list;
- device info;
- session start;
- ps for Windows PowerShell;
- cmd for Windows CMD;
- sh for Linux shell;
- job status/result;
- reboot with explicit confirmation;
- --json output;
- --save local artifacts;
- stable exit codes.
```

### Client Agent

```text
- Go binary;
- Windows Service support;
- systemd daemon support;
- enrollment;
- config file;
- secure outbound connection;
- heartbeat;
- capabilities reporting;
- shell.exec jobs;
- command timeout;
- stdout/stderr size limits;
- reboot job;
- reconnect after network loss;
- startup after reboot;
- local structured logs.
```

## Out of Scope for MVP v0

```text
- remote desktop;
- screen sharing;
- interactive terminal;
- full web UI;
- multi-tenant billing;
- SSO;
- complex RBAC;
- file manager UI;
- auto-update;
- mobile clients;
- macOS agent;
- unattended hidden install;
- EDR/antivirus bypass;
- credential extraction;
- keylogging.
```

## MVP Milestones

### Milestone 1: Protocol and Skeleton

```text
- monorepo structure;
- protocol schemas;
- server bootstrap;
- CLI bootstrap;
- Go agent bootstrap;
- CI test commands.
```

### Milestone 2: Enrollment and Heartbeat

```text
- create enrollment token;
- agent enroll;
- device identity created;
- agent connects;
- server shows online status.
```

### Milestone 3: Shell Job Execution

```text
- create shell.exec job from CLI;
- route job to agent;
- execute command;
- return result;
- persist result;
- output as JSON.
```

### Milestone 4: Debug Sessions

```text
- session start;
- attach jobs to session;
- save local artifacts;
- create basic markdown report.
```

### Milestone 5: Reboot and Reconnect

```text
- reboot job with confirmation;
- device state: rebooting;
- reconnect after startup;
- audit event.
```
