CREATE TABLE IF NOT EXISTS administrators (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS enrollment_tokens (
  id TEXT PRIMARY KEY,
  token_hash TEXT NOT NULL UNIQUE,
  name_hint TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  used_by_device_id TEXT,
  revoked_at TIMESTAMPTZ,
  created_by_admin_id TEXT REFERENCES administrators(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS devices (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  status TEXT NOT NULL,
  os_type TEXT NOT NULL,
  os_version TEXT NOT NULL,
  distro TEXT,
  agent_version TEXT NOT NULL,
  protocol_version TEXT NOT NULL,
  capabilities_json JSONB NOT NULL,
  secret_hash TEXT NOT NULL,
  last_seen_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  device_id TEXT NOT NULL REFERENCES devices(id),
  name TEXT NOT NULL,
  status TEXT NOT NULL,
  created_by_admin_id TEXT REFERENCES administrators(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  closed_at TIMESTAMPTZ,
  summary TEXT
);

CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  session_id TEXT REFERENCES sessions(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  created_by_admin_id TEXT REFERENCES administrators(id),
  kind TEXT NOT NULL,
  risk_level TEXT NOT NULL,
  status TEXT NOT NULL,
  payload_json JSONB NOT NULL,
  timeout_sec INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  dispatched_at TIMESTAMPTZ,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS job_results (
  id TEXT PRIMARY KEY,
  job_id TEXT NOT NULL UNIQUE REFERENCES jobs(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  status TEXT NOT NULL,
  exit_code INTEGER,
  stdout TEXT NOT NULL,
  stderr TEXT NOT NULL,
  stdout_truncated BOOLEAN NOT NULL,
  stderr_truncated BOOLEAN NOT NULL,
  error_code TEXT,
  error_message TEXT,
  started_at TIMESTAMPTZ NOT NULL,
  finished_at TIMESTAMPTZ NOT NULL,
  duration_ms INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_events (
  id TEXT PRIMARY KEY,
  admin_id TEXT REFERENCES administrators(id),
  device_id TEXT REFERENCES devices(id),
  session_id TEXT REFERENCES sessions(id),
  job_id TEXT REFERENCES jobs(id),
  action TEXT NOT NULL,
  risk_level TEXT,
  result TEXT NOT NULL,
  metadata_json JSONB NOT NULL,
  request_id TEXT,
  source_ip TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO administrators (id, name, status)
VALUES ('admin_mvp', 'admin_mvp', 'active')
ON CONFLICT (id) DO NOTHING;
