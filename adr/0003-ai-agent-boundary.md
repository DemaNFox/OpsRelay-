# ADR 0003: AI-Agent Boundary

## Status

Accepted for MVP planning.

## Context

The project is intended for AI-assisted remote diagnostics. However, running AI logic directly on client machines would increase risk and complexity.

## Decision

AI-agent runs only on the administrator machine. The client-agent only executes explicit jobs and returns results.

## Rationale

This keeps:

```text
- reasoning local to administrator;
- client-agent minimal and auditable;
- protocol simple;
- client device free from prompt/context storage;
- AI-agent replaceable.
```

## Consequences

Codex, Claude Code or another AI-agent should interact only through `remote-admin` CLI/API.
