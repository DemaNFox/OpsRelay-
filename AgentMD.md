# AgentMD

This file is the primary instruction source for AI/code agents working on OpsRelay.

## Project Boundary

OpsRelay is an authorized remote administration MVP. It is not malware, a backdoor, a stealth tool, or an unauthorized access mechanism.

Core boundary:

```text
AI-agent runs only on the administrator machine.
Server routes, authenticates, authorizes and audits.
Client agent executes explicit authorized jobs and returns structured results.
Client machine does not run AI reasoning.
```

## Architecture

Keep components separated:

```text
apps/server       Node.js + TypeScript server API, relay, jobs, sessions, audit
apps/admin-cli    Node.js + TypeScript administrator CLI
agent             Go Windows/Linux client agent
packages/protocol shared protocol contracts
docs              product, architecture and operational documentation
adr               architecture decisions
```

Do not mix server, CLI, protocol and agent logic into a single monolith.

## Security Rules

Required:

```text
- administrator API token;
- one-time enrollment token;
- device identity after enrollment;
- audit log for jobs and sensitive operations;
- command timeout;
- stdout/stderr size limits;
- explicit confirmation for reboot/destructive operations;
- visible client agent behavior;
- no secrets in logs or committed files.
```

Forbidden:

```text
- stealth installation;
- hiding process/files/network traffic;
- AV/EDR bypass;
- credential dumping;
- keylogging;
- unauthorized screen capture;
- privilege escalation bypass;
- direct AI connection to client devices;
- destructive action without administrator confirmation.
```

## TypeScript Style

Use strict TypeScript. Avoid `any`. Validate runtime inputs with protocol schemas or local validators. Keep feature boundaries:

```text
domain          models and business rules
application     use cases/services
infrastructure  database, transport and external integrations
presentation    HTTP/WebSocket/CLI handlers
```

Presentation code must not directly contain persistence or business policy logic.

## Go Agent Style

The Go agent must keep core logic separate from platform adapters:

```text
cmd/opsrelay-agent
internal/core
internal/transport
internal/executor
internal/platform
internal/config
internal/logging
```

Core should depend on interfaces where practical. Shell execution must enforce timeout and output limits.

## Protocol Rules

Protocol changes must remain explicit and versioned. Shared contracts live in `packages/protocol`.

Every job/result flow must include:

```text
device_id
session_id when available
job_id
status
stdout
stderr
exit_code
timestamps
duration_ms
truncation flags
```

## CLI Rules

Admin CLI commands used by automation must support:

```text
--json
--session <session-id>
--timeout <seconds>
--save
stable exit codes
```

Machine-readable errors must use:

```json
{
  "ok": false,
  "error": {
    "code": "DEVICE_OFFLINE",
    "message": "Device is offline",
    "retryable": true
  }
}
```

## Testing

Minimum checks before considering work complete:

```bash
pnpm typecheck
pnpm test
pnpm build

cd agent
go test ./...
go vet ./...
```

Feature work should include focused tests and documentation updates.

## Definition of Done

A feature is complete only when:

```text
- strict types/contracts exist;
- tests pass;
- errors are normalized;
- structured outputs are machine-readable;
- docs are updated;
- secrets are not committed;
- code follows component boundaries;
- local run/test commands are documented.
```
