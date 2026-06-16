# Server API

## Responsibility

The server is the control plane. It owns identity, authorization, device connectivity, job lifecycle and audit history.

## Recommended Stack

```text
Runtime:     Node.js LTS
Language:    TypeScript strict mode
Framework:   Fastify or NestJS
Database:    PostgreSQL
ORM/Query:   Prisma, Drizzle, or Kysely
Transport:   WebSocket or gRPC streaming for agents
Validation:  Zod, TypeBox, Valibot, or generated protocol validators
Logging:     pino or OpenTelemetry-compatible logger
```

Fastify is preferred for a lightweight modular MVP. NestJS is acceptable if the team wants a more opinionated enterprise structure.

## Main Modules

```text
modules/auth
modules/devices
modules/enrollment
modules/agent-gateway
modules/jobs
modules/sessions
modules/audit
modules/policies
```

## API Surfaces

### Admin HTTP API

Used by CLI and future web UI.

```text
POST   /v1/auth/token                 optional if using static admin token for MVP
POST   /v1/enrollment-tokens
GET    /v1/devices
GET    /v1/devices/:deviceId
POST   /v1/sessions
GET    /v1/sessions/:sessionId
POST   /v1/jobs
GET    /v1/jobs/:jobId
GET    /v1/jobs/:jobId/result
POST   /v1/devices/:deviceId/reboot
GET    /v1/audit
```

### Agent Gateway

Used by client-agent.

```text
POST   /v1/agent/enroll
WS     /v1/agent/connect
```

or gRPC equivalents:

```text
AgentService.Enroll
AgentService.Connect(stream AgentMessage) returns (stream ServerMessage)
```

## Job Lifecycle

```text
queued → dispatched → running → completed
queued → dispatched → running → failed
queued → expired
running → timed_out
running → interrupted_by_reboot → waiting_reconnect → failed/resumed
```

For MVP, actual resume is optional. State tracking is required.

## Server-Side Validation

The server must validate:

```text
- admin authentication;
- admin permission for target device;
- device exists;
- device is online or job can be queued;
- job kind is supported by target capabilities;
- timeout is within allowed limits;
- command size is within allowed limits;
- reboot/destructive actions require explicit confirmation.
```

## Error Format

All admin API errors should map to this shape:

```json
{
  "ok": false,
  "error": {
    "code": "DEVICE_OFFLINE",
    "message": "Device is offline",
    "retryable": true,
    "details": {}
  }
}
```

## Audit Requirements

Every sensitive operation must create an audit event:

```text
- enrollment token created;
- device enrolled;
- admin listed devices;
- job created;
- job dispatched;
- job completed/failed;
- reboot requested;
- device revoked;
- token revoked.
```

## Development Commands

Planned commands:

```bash
pnpm --filter @remote-admin/server dev
pnpm --filter @remote-admin/server test
pnpm --filter @remote-admin/server lint
pnpm --filter @remote-admin/server typecheck
pnpm --filter @remote-admin/server db:migrate
```
