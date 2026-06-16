# Data Model

## Overview

The initial database can be PostgreSQL. The data model should stay simple for MVP but leave room for organizations, RBAC and device groups.

## Core Tables

### administrators

```text
id
email/name
status
created_at
updated_at
```

For MVP, this can be simplified if using a static admin token.

### admin_tokens

```text
id
admin_id
name
token_hash
scopes
last_used_at
expires_at
revoked_at
created_at
```

Never store raw tokens.

### devices

```text
id
name
status
os_type
os_version
distro
agent_version
protocol_version
capabilities_json
last_seen_at
revoked_at
created_at
updated_at
```

### enrollment_tokens

```text
id
token_hash
name_hint
expires_at
used_at
used_by_device_id
revoked_at
created_by_admin_id
created_at
```

### agent_connections

```text
id
device_id
connected_at
disconnected_at
remote_addr
agent_version
protocol_version
last_heartbeat_at
```

### sessions

```text
id
device_id
name
status
created_by_admin_id
created_at
closed_at
summary
```

### jobs

```text
id
session_id
device_id
created_by_admin_id
kind
risk_level
status
payload_json
timeout_sec
created_at
dispatched_at
started_at
finished_at
expires_at
```

### job_results

```text
id
job_id
exit_code
stdout
stderr
stdout_truncated
stderr_truncated
error_code
error_message
duration_ms
created_at
```

For large outputs, later move stdout/stderr to object storage.

### audit_events

```text
id
admin_id
device_id
session_id
job_id
action
risk_level
result
metadata_json
request_id
source_ip
created_at
```

## Status Enums

### device.status

```text
offline
online
rebooting
revoked
unknown
```

### job.status

```text
queued
dispatched
running
completed
failed
timed_out
expired
cancelled
interrupted_by_reboot
```

### session.status

```text
open
closed
failed
```

## Migration Requirements

Migrations must be versioned and reproducible.

Commands should exist for:

```bash
pnpm --filter @remote-admin/server db:migrate
pnpm --filter @remote-admin/server db:rollback
pnpm --filter @remote-admin/server db:seed
```
