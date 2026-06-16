# Non-Goals and Constraints

## Non-Goals for MVP

The MVP must not attempt to solve every RMM problem.

Out of scope:

```text
- remote desktop;
- hidden/silent deployment;
- screen capture;
- keylogging;
- credential extraction;
- AV/EDR bypass;
- malware-like persistence;
- peer-to-peer NAT traversal;
- mobile apps;
- complex enterprise RBAC;
- custom scripting language;
- full package management;
- full file manager UI;
- full web admin panel.
```

## Constraints

```text
- AI-agent must run only on administrator machine.
- Client-agent must not contain AI reasoning logic.
- Client-agent must connect outbound to server.
- All command executions must be auditable.
- All jobs must have timeout and output limits.
- Destructive actions must require confirmation/policy approval.
- System must be AI-agent agnostic.
- Protocol must support Windows and Linux from the start.
```

## Design Boundaries

### The Server Is Not the Executor

Server routes commands and stores results. It must not run client diagnostics locally.

### The CLI Is Not a Bypass

CLI must use official server API only. It must not include hidden direct access paths.

### The Agent Is Not Autonomous

Agent receives jobs and returns results. It should not decide diagnostics on its own.

## Security Boundary

The platform must be designed for explicit authorized administration. Any feature that makes the agent hidden, deceptive or evasive is outside the product scope.
