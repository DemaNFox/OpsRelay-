# Security and Authorization

## Security Model

Remote Admin MVP gives powerful administration capabilities. The system must be designed as an authorized, auditable RMM-like platform, not as an unrestricted hidden remote shell.

## Minimum Security Requirements

```text
- TLS for all connections;
- administrator authentication;
- one-time enrollment token;
- device identity after enrollment;
- device revocation;
- audit log for every job;
- command timeouts;
- output size limits;
- destructive action confirmation;
- no secrets in logs;
- no hidden installation behavior.
```

## Identity Model

### Administrator Identity

For MVP, administrator auth can start with API tokens.

Future:

```text
- OIDC/SSO;
- passkeys;
- TOTP/2FA;
- RBAC;
- scoped tokens.
```

### Device Identity

A device must not be trusted only by human-readable ID.

Enrollment flow:

```text
1. Admin creates one-time enrollment token.
2. Agent calls enroll endpoint with token and device metadata.
3. Server creates device record.
4. Server returns device identity material.
5. Token is consumed and cannot be reused.
6. Future connections authenticate as that device.
```

## Authorization Model

For MVP:

```text
admin token → allowed to manage all devices in one organization
```

Future:

```text
organization → teams → roles → device groups → capabilities
```

Potential permissions:

```text
devices.read
jobs.create.readonly
jobs.create.write
jobs.reboot
files.read
files.write
sessions.manage
audit.read
```

## Command Risk Levels

### Level 1: Read-Only Diagnostics

Can be allowed for AI-agent automatic execution:

```text
Get-Service
Get-Process
Get-ComputerInfo
systeminfo
ipconfig /all
systemctl status
journalctl read-only queries
```

### Level 2: Controlled Changes

Require explicit confirmation:

```text
Restart-Service
systemctl restart
Copy-Item
New-Item
Remove-Item
Stop-Process
package install/remove
reboot
```

### Level 3: High-Risk / Blocked by Default

Blocked or requires special policy:

```text
format disk/volume
remove users
disable security tools
delete backups/snapshots
modify boot configuration
credential extraction
```

## Secrets Handling

Never log:

```text
- admin tokens;
- enrollment tokens;
- device secrets;
- private keys;
- passwords;
- full command output known to contain secrets.
```

Use masking helpers for logs and audit data.

## Audit Events

Audit events should include:

```text
timestamp
admin_id
device_id
session_id
job_id
action
result
source_ip/request_id
risk_level
```

## Explicitly Prohibited Features

The project must not implement:

```text
- stealth install;
- hidden persistence;
- AV/EDR bypass;
- credential dumping;
- keylogging;
- unauthorized surveillance;
- privilege escalation bypass;
- any feature intended to evade user or system visibility.
```

## Secure Defaults

```text
- deny unknown job kinds;
- deny unsupported capabilities;
- timeout required for every job;
- max output size required;
- destructive actions require confirmation;
- offline devices do not receive stale dangerous jobs by default;
- revoked devices cannot reconnect.
```
