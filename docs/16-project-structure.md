# Project Structure

## Recommended Monorepo Layout

```text
remote-admin/
├─ README.md
├─ AgentMD.md
├─ AGENTS.md
├─ docs/
├─ adr/
├─ apps/
│  ├─ server/
│  ├─ admin-cli/
│  └─ admin-web/
├─ packages/
│  ├─ protocol/
│  ├─ sdk-js/
│  └─ shared/
├─ agent/
│  ├─ cmd/
│  │  └─ remote-agent/
│  ├─ internal/
│  │  ├─ core/
│  │  ├─ transport/
│  │  ├─ executor/
│  │  ├─ platform/
│  │  │  ├─ windows/
│  │  │  └─ linux/
│  │  ├─ config/
│  │  ├─ logging/
│  │  └─ security/
│  └─ packaging/
│     ├─ windows/
│     └─ linux/
└─ infra/
   ├─ docker-compose.yml
   ├─ caddy/
   └─ postgres/
```

## TypeScript Workspace

Recommended package manager:

```text
pnpm workspaces
```

Root scripts:

```json
{
  "scripts": {
    "lint": "pnpm -r lint",
    "typecheck": "pnpm -r typecheck",
    "test": "pnpm -r test",
    "build": "pnpm -r build"
  }
}
```

## Server Structure

```text
apps/server/src/
  app/
  config/
  shared/
  modules/
    auth/
    devices/
    enrollment/
    agent-gateway/
    jobs/
    sessions/
    audit/
    policies/
```

Each module:

```text
domain/
application/
infrastructure/
presentation/
tests/
```

## CLI Structure

```text
apps/admin-cli/src/
  app/
  config/
  client/
  commands/
    devices/
    device/
    session/
    shell/
    job/
    reboot/
  output/
  session-artifacts/
  errors/
```

## Protocol Package

```text
packages/protocol/
  schemas/
  examples/
  generated/
  tests/
```

The protocol package should be dependency-light and should not depend on server implementation.

## Go Agent Structure

```text
agent/
  go.mod
  cmd/remote-agent/main.go
  internal/core/
  internal/transport/
  internal/executor/
  internal/platform/
    windows/
    linux/
  internal/config/
  internal/logging/
  internal/security/
  packaging/
```

## Naming Rules

```text
Project binary CLI: remote-admin
Agent binary:        remote-agent
Windows service:     RemoteAdminAgent
Linux service:       remote-agent.service
```
