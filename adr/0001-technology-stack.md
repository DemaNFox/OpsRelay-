# ADR 0001: Technology Stack

## Status

Accepted for MVP planning.

## Context

The project needs a server, admin CLI, future web UI and a cross-platform client-agent for Windows and Linux/Ubuntu.

The primary developer experience is Node.js/TypeScript, but the client-agent must behave like a native service/daemon and run reliably after reboot.

## Decision

Use:

```text
Server:       Node.js + TypeScript
Admin CLI:    Node.js + TypeScript
Admin Web:    Next.js + TypeScript, later
Client Agent: Go
Protocol:     JSON Schema/OpenAPI for MVP, Protobuf/gRPC possible later
Database:     PostgreSQL
```

## Rationale

TypeScript is a strong fit for server, CLI and web layers.

Go is a strong fit for client-agent because it provides:

```text
- single binary builds;
- cross-platform support;
- good daemon/service behavior;
- simple deployment;
- strong standard library for networking/processes;
- good reliability profile.
```

## Consequences

The protocol must be language-neutral. Types should be generated from shared contracts rather than copied manually between TypeScript and Go.
