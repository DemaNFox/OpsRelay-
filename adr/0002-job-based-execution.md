# ADR 0002: Job-Based Execution First

## Status

Accepted for MVP planning.

## Context

Interactive terminals are convenient but hard to audit, test, replay, limit and integrate with AI-agents.

## Decision

MVP will use job-based execution:

```text
CLI creates job → server dispatches job → agent executes job → result is returned and persisted
```

Interactive terminal is explicitly out of MVP scope.

## Rationale

Job-based execution improves:

```text
- auditability;
- deterministic outputs;
- timeout handling;
- AI-agent parsing;
- local artifact storage;
- retry/reconnect design;
- approval workflows.
```

## Consequences

All CLI commands should map to jobs. Session artifacts should persist every command and result.
