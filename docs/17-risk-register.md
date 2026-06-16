# Risk Register

## Security Risks

### Risk: Remote shell can be abused

Mitigation:

```text
- authorized enrollment only;
- audit log;
- admin auth;
- destructive action approval;
- device revocation;
- command timeout/output limits;
- no stealth features.
```

### Risk: Device token leakage

Mitigation:

```text
- one-time enrollment token;
- token hashing server-side;
- secure storage for device identity;
- revoke device;
- rotate credentials later.
```

### Risk: AI-agent performs unsafe change

Mitigation:

```text
- read-only first workflow;
- command risk levels;
- explicit approval;
- CLI guardrails;
- audit records.
```

## Reliability Risks

### Risk: Client behind NAT/offline

Mitigation:

```text
- outbound persistent connection;
- heartbeat;
- online/offline state;
- retry/backoff;
- clear CLI errors.
```

### Risk: Reboot interrupts job

Mitigation:

```text
- rebooting state;
- agent service auto-start;
- reconnect handling;
- job state transition interrupted_by_reboot.
```

### Risk: Command produces huge output

Mitigation:

```text
- stdout/stderr limits;
- truncation flags;
- future object storage for large outputs.
```

## Product Risks

### Risk: MVP scope expands too much

Mitigation:

```text
- job-based shell only first;
- no remote desktop in MVP;
- no full web UI in MVP;
- prioritize protocol/server/agent/CLI.
```

### Risk: Tied too closely to one AI-agent

Mitigation:

```text
- stable CLI/API;
- JSON output;
- no Codex-specific protocol;
- SDK later.
```

## Technical Risks

### Risk: Node.js agent is less production-friendly

Mitigation:

```text
- use Go for client-agent;
- keep TypeScript for server/CLI/UI;
- protocol contracts isolate implementation.
```

### Risk: Windows/Linux behavior diverges

Mitigation:

```text
- platform adapters;
- capabilities model;
- OS-aware CLI commands;
- contract tests.
```
