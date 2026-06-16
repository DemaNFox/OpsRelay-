# AI-Agent Workflow

## Purpose

This document defines how Codex, Claude Code or another AI-agent should interact with the system.

## Key Rule

The AI-agent must run on the administrator machine and use only `remote-admin` CLI or official SDK/API. The client machine should only receive explicit execution jobs.

## Correct Workflow

```text
1. Understand task.
2. Query device info.
3. Start debug session.
4. Run read-only diagnostics.
5. Analyze results locally.
6. Propose fix.
7. Ask for approval if fix changes system state.
8. Execute approved action.
9. Verify result.
10. Generate report.
```

## Example: Windows Printer Debug

```bash
remote-admin device info win-office-01 --json
remote-admin session start win-office-01 --name printer-debug --json
remote-admin ps win-office-01 "Get-Service Spooler" --json --save
remote-admin ps win-office-01 "Get-Printer" --json --save
remote-admin ps win-office-01 "Get-WinEvent -LogName System -MaxEvents 50" --json --save
```

If restart is needed:

```bash
remote-admin ps win-office-01 "Restart-Service Spooler" --json --save
```

This should require approval according to policy.

## Example: Ubuntu Service Debug

```bash
remote-admin device info ubuntu-01 --json
remote-admin session start ubuntu-01 --name nginx-debug --json
remote-admin sh ubuntu-01 "systemctl status nginx --no-pager" --json --save
remote-admin sh ubuntu-01 "journalctl -u nginx --since '2 hours ago' --no-pager" --json --save
remote-admin sh ubuntu-01 "ss -tulpn | grep ':80\|:443'" --json --save
```

If restart is needed:

```bash
remote-admin sh ubuntu-01 "sudo systemctl restart nginx" --json --save
```

This should require approval according to policy.

## AI-Agent Prompt Contract

Recommended system/project instruction for the AI-agent:

```text
Use only remote-admin CLI for remote diagnostics.
Never connect to client devices directly.
Always start with device info and a debug session.
Prefer read-only commands first.
Do not execute destructive or state-changing operations without explicit approval.
Save outputs into the active session.
Analyze results locally.
Generate a concise report at the end.
```

## Local Session Context

AI-agent should use files saved under `.sessions/<session-id>/` as the working context.

Recommended artifacts:

```text
session.json
transcript.md
commands/*.command.*
results/*.stdout.txt
results/*.stderr.txt
results/*.meta.json
reports/final.md
```

## Replacing Codex

The architecture must remain AI-agent agnostic:

```text
Codex
Claude Code
Cursor Agent
OpenAI Agent
local LLM agent
human operator
scripts/CI
```

All of them should work through the same CLI/API.
