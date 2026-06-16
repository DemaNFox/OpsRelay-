# Testing Strategy

## Philosophy

The platform performs remote administration, so correctness, safety and observability are mandatory. Every feature must have tests appropriate to its risk level.

## TypeScript Components

Required checks:

```bash
pnpm lint
pnpm typecheck
pnpm test
```

### Unit Tests

Cover:

```text
- domain models;
- policy decisions;
- job state transitions;
- error mapping;
- config validation;
- CLI output formatting.
```

### Integration Tests

Cover:

```text
- API endpoints;
- database repositories;
- enrollment flow;
- job creation and result persistence;
- audit events.
```

Use test database containers where practical.

### Contract Tests

Protocol schemas must be tested against known examples:

```text
- valid heartbeat;
- valid shell.exec request;
- valid job result;
- invalid unsupported shell;
- incompatible protocol version.
```

## Go Agent

Required checks:

```bash
go test ./...
go vet ./...
```

### Unit Tests

Cover:

```text
- config parsing;
- reconnect backoff;
- shell executor timeout;
- stdout/stderr truncation;
- job lifecycle;
- platform interface behavior using fakes.
```

### Platform Tests

Platform-specific tests should be isolated.

```text
Windows tests: services, PowerShell executor, reboot adapter mocked where needed.
Linux tests: systemd/journald adapters mocked or run in container where safe.
```

Do not run destructive reboot tests in normal CI.

## End-to-End Smoke Tests

Minimum e2e scenario:

```text
1. Start server with test DB.
2. Create enrollment token.
3. Start test agent.
4. Agent enrolls and connects.
5. CLI lists device online.
6. CLI sends shell.exec.
7. Agent returns result.
8. Server stores result.
9. CLI writes local session artifact.
```

## Security Tests

Required cases:

```text
- invalid admin token rejected;
- expired enrollment token rejected;
- reused enrollment token rejected;
- revoked device rejected;
- unsupported capability rejected;
- destructive job without confirmation rejected;
- secrets masked in logs.
```

## Definition of Done for Tests

A feature is not complete unless:

```text
- tests pass locally;
- type checks pass;
- at least one negative/error path is tested;
- docs or examples are updated;
- no test relies on external hidden state.
```
