# Observability and Audit

## Goals

The system must be easy to debug and safe to operate. Every command should be traceable from administrator request to client execution and result.

## Correlation IDs

Every request/job should carry:

```text
request_id
session_id
job_id
device_id
admin_id
connection_id
```

## Structured Logs

All components should use structured logs.

Example server log:

```json
{
  "level": "info",
  "msg": "job completed",
  "request_id": "req_123",
  "session_id": "ses_123",
  "job_id": "job_123",
  "device_id": "dev_123",
  "status": "completed",
  "duration_ms": 820
}
```

## Logs by Component

### Server

```text
- API requests;
- auth decisions;
- device connect/disconnect;
- heartbeat;
- job lifecycle;
- audit events;
- server errors.
```

### Admin CLI

```text
- command invocation;
- server request/response metadata;
- local artifact write paths;
- user-facing errors.
```

### Client Agent

```text
- startup/shutdown;
- enrollment;
- connect/reconnect;
- heartbeat;
- job received;
- command started/finished;
- timeout/cancellation;
- reboot requested;
- local errors.
```

## Secrets in Logs

Never log raw:

```text
tokens
passwords
private keys
full auth headers
secrets embedded in env vars
```

## Audit vs Logs

Logs are operational and may be noisy. Audit events are durable security records.

Audit must record:

```text
who did what, where, when, through which session, with what result
```

## Metrics for Future

```text
online devices count
job success/failure rate
job duration histogram
agent reconnect count
heartbeat latency
server gateway connections
command timeout count
```

## Tracing for Future

OpenTelemetry can be added later. The MVP should still preserve correlation IDs to make tracing easy.
