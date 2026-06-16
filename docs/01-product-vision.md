# Product Vision

## Summary

Remote Admin MVP is an authorized remote administration platform designed for AI-assisted debugging and diagnostics of client devices.

The platform allows an administrator to run an AI-agent locally on the administrator machine, send diagnostic jobs through a controlled CLI/API, execute those jobs on a remote Windows/Linux client through a lightweight installed agent, and receive structured outputs back for local analysis.

## Main Goal

Enable this workflow:

```text
Administrator asks AI-agent:
"Debug printer issue on client-001."

AI-agent uses remote-admin CLI.
CLI sends jobs to server.
Server routes jobs to client-agent.
Client-agent runs PowerShell/CMD/sh commands.
Results return to administrator machine.
AI-agent analyzes results locally and proposes next steps.
```

## Core Principles

```text
- AI reasoning stays on the administrator machine.
- Client machine only executes explicit jobs.
- Server controls auth, routing, audit and policy.
- Client-agent is cross-platform: Windows first, Linux/Ubuntu planned from the start.
- AI-agent is replaceable: Codex, Claude Code, another agent or a human can use the same CLI/API.
```

## Target Users

```text
- IT administrators
- MSP/RMM-style operators
- internal support engineers
- developers debugging customer environments with authorization
```

## Authorized Use Requirement

The platform is intended only for devices where the operator has explicit permission to install and operate the agent. Product design must always include visible installation, device enrollment, audit logging and access revocation.

## MVP Success Criteria

MVP is successful when an administrator can:

```text
1. Install client-agent on Windows and Ubuntu.
2. Enroll device into server.
3. See device online from CLI.
4. Start a debug session.
5. Execute PowerShell/CMD/sh commands.
6. Receive stdout/stderr/exit_code.
7. Persist command history and outputs locally.
8. Reboot device with confirmation.
9. See audit history on the server.
10. Use Codex/another AI-agent through the same CLI.
```
