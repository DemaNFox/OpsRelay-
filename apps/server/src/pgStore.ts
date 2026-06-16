import { randomUUID } from "node:crypto";
import pg from "pg";
import {
  PROTOCOL_VERSION,
  capabilityForShell,
  jobToAgentJob,
  type AgentEnrollRequest,
  type Capability,
  type Device,
  type Job,
  type JobCreateRequest,
  type JobForAgent,
  type JobKind,
  type JobResult,
  type RiskLevel,
  type Session
} from "@opsrelay/protocol";
import { createSecret, sha256 } from "./crypto.js";
import { badRequest, conflict, forbidden, notFound } from "./errors.js";
import type { AuditEvent, Store, StoreSnapshot } from "./store.js";

const { Pool } = pg;

interface EnrollmentRow {
  id: string;
  token_hash: string;
  name_hint: string;
  expires_at: Date;
  used_at: Date | null;
  used_by_device_id: string | null;
}

interface DeviceRow {
  id: string;
  name: string;
  status: Device["status"];
  os_type: Device["os"]["type"];
  os_version: string;
  distro: string | null;
  agent_version: string;
  protocol_version: string;
  capabilities_json: Capability[];
  secret_hash: string;
  last_seen_at: Date | null;
}

interface SessionRow {
  id: string;
  device_id: string;
  name: string;
  status: Session["status"];
  created_at: Date;
  closed_at: Date | null;
}

interface JobPayload {
  shell: Job["shell"];
  command: string | null;
  stdout_limit_bytes: number;
  stderr_limit_bytes: number;
  created_by: string;
}

interface JobRow {
  id: string;
  session_id: string | null;
  device_id: string;
  created_by_admin_id: string;
  kind: Job["kind"];
  risk_level: RiskLevel;
  status: Job["status"];
  payload_json: JobPayload;
  timeout_sec: number;
  created_at: Date;
  dispatched_at: Date | null;
  started_at: Date | null;
  finished_at: Date | null;
}

interface JobResultRow {
  job_id: string;
  device_id: string;
  status: "completed" | "failed" | "timed_out";
  exit_code: number | null;
  stdout: string;
  stderr: string;
  stdout_truncated: boolean;
  stderr_truncated: boolean;
  error_code: string | null;
  error_message: string | null;
  started_at: Date;
  finished_at: Date;
  duration_ms: number;
}

interface AuditRow {
  id: string;
  admin_id: string | null;
  device_id: string | null;
  session_id: string | null;
  job_id: string | null;
  action: string;
  risk_level: RiskLevel | null;
  result: string;
  metadata_json: Record<string, unknown>;
  created_at: Date;
}

export class PostgresStore implements Store {
  private readonly pool: pg.Pool;

  private constructor(pool: pg.Pool) {
    this.pool = pool;
  }

  public static async connect(databaseUrl: string): Promise<PostgresStore> {
    const pool = new Pool({ connectionString: databaseUrl });
    await pool.query("select 1");
    return new PostgresStore(pool);
  }

  public async close(): Promise<void> {
    await this.pool.end();
  }

  public async createEnrollmentToken(name: string, ttlSec: number, adminId: string): Promise<{ id: string; token: string; expires_at: string }> {
    await this.ensureAdmin(adminId);
    const id = this.id("enr");
    const token = createSecret("enr");
    const expiresAt = new Date(Date.now() + ttlSec * 1000);
    await this.pool.query(
      `insert into enrollment_tokens (id, token_hash, name_hint, expires_at, created_by_admin_id)
       values ($1, $2, $3, $4, $5)`,
      [id, sha256(token), name, expiresAt, adminId]
    );
    await this.audit(adminId, null, null, null, "enrollment_token.created", "low", "success", { enrollment_token_id: id, name });
    return { id, token, expires_at: expiresAt.toISOString() };
  }

  public async enrollDevice(request: AgentEnrollRequest): Promise<{ device: Device; device_secret: string }> {
    if (request.protocol_version !== PROTOCOL_VERSION) {
      throw badRequest("INCOMPATIBLE_PROTOCOL", "Agent protocol version is not supported", {
        expected: PROTOCOL_VERSION,
        actual: request.protocol_version
      });
    }
    const tokenHash = sha256(request.token);
    const tokenResult = await this.pool.query<EnrollmentRow>("select * from enrollment_tokens where token_hash = $1", [tokenHash]);
    const tokenRecord = tokenResult.rows[0];
    if (tokenRecord === undefined) {
      throw forbidden("INVALID_ENROLLMENT_TOKEN", "Enrollment token is invalid");
    }
    if (tokenRecord.used_at !== null) {
      throw forbidden("ENROLLMENT_TOKEN_USED", "Enrollment token has already been used");
    }
    if (tokenRecord.expires_at.getTime() <= Date.now()) {
      throw forbidden("ENROLLMENT_TOKEN_EXPIRED", "Enrollment token has expired");
    }

    const deviceId = this.id("dev");
    const secret = createSecret("devsec");
    await this.pool.query("begin");
    try {
      await this.pool.query(
        `insert into devices (
          id, name, status, os_type, os_version, distro, agent_version, protocol_version, capabilities_json, secret_hash, last_seen_at
        ) values ($1, $2, 'offline', $3, $4, $5, $6, $7, $8, $9, null)`,
        [
          deviceId,
          request.name,
          request.os.type,
          request.os.version,
          request.os.distro ?? null,
          request.agent_version,
          request.protocol_version,
          JSON.stringify(request.capabilities),
          sha256(secret)
        ]
      );
      await this.pool.query("update enrollment_tokens set used_at = now(), used_by_device_id = $1 where id = $2", [deviceId, tokenRecord.id]);
      await this.pool.query("commit");
    } catch (error) {
      await this.pool.query("rollback");
      throw error;
    }

    const device = await this.getDevice(deviceId);
    await this.audit(null, deviceId, null, null, "device.enrolled", "low", "success", {
      enrollment_token_id: tokenRecord.id,
      device_name: request.name
    });
    return { device, device_secret: secret };
  }

  public async authenticateDevice(deviceId: string, deviceSecret: string): Promise<Device> {
    const row = await this.getDeviceRow(deviceId);
    if (row.secret_hash !== sha256(deviceSecret)) {
      throw forbidden("INVALID_DEVICE_IDENTITY", "Device identity is invalid");
    }
    if (row.status === "revoked") {
      throw forbidden("DEVICE_REVOKED", "Device has been revoked");
    }
    return this.mapDevice(row);
  }

  public async markDeviceOnline(deviceId: string): Promise<Device> {
    await this.pool.query("update devices set status = 'online', last_seen_at = now(), updated_at = now() where id = $1", [deviceId]);
    await this.audit(null, deviceId, null, null, "device.connected", "low", "success", {});
    return this.getDevice(deviceId);
  }

  public async markDeviceOffline(deviceId: string): Promise<void> {
    await this.pool.query("update devices set status = 'offline', last_seen_at = now(), updated_at = now() where id = $1 and status <> 'revoked'", [deviceId]);
    await this.audit(null, deviceId, null, null, "device.disconnected", "low", "success", {});
  }

  public async updateHeartbeat(deviceId: string, agentVersion: string): Promise<Device> {
    await this.pool.query("update devices set status = 'online', agent_version = $2, last_seen_at = now(), updated_at = now() where id = $1", [
      deviceId,
      agentVersion
    ]);
    return this.getDevice(deviceId);
  }

  public async listDevices(adminId: string): Promise<Device[]> {
    await this.ensureAdmin(adminId);
    await this.audit(adminId, null, null, null, "devices.listed", "low", "success", {});
    const result = await this.pool.query<DeviceRow>("select * from devices order by created_at desc");
    return result.rows.map((row) => this.mapDevice(row));
  }

  public async getDevice(deviceId: string): Promise<Device> {
    return this.mapDevice(await this.getDeviceRow(deviceId));
  }

  public async createSession(deviceId: string, name: string, adminId: string): Promise<Session> {
    await this.ensureAdmin(adminId);
    await this.getDevice(deviceId);
    const id = this.id("ses");
    const result = await this.pool.query<SessionRow>(
      `insert into sessions (id, device_id, name, status, created_by_admin_id)
       values ($1, $2, $3, 'open', $4)
       returning id, device_id, name, status, created_at, closed_at`,
      [id, deviceId, name, adminId]
    );
    const row = this.expectRow(result.rows[0], "SESSION_NOT_CREATED", "Session was not created");
    await this.audit(adminId, deviceId, row.id, null, "session.created", "low", "success", { name });
    return this.mapSession(row);
  }

  public async getSession(sessionId: string): Promise<Session> {
    const result = await this.pool.query<SessionRow>("select id, device_id, name, status, created_at, closed_at from sessions where id = $1", [sessionId]);
    const row = result.rows[0];
    if (row === undefined) {
      throw notFound("SESSION_NOT_FOUND", "Session was not found");
    }
    return this.mapSession(row);
  }

  public async createJob(request: JobCreateRequest, adminId: string): Promise<Job> {
    await this.ensureAdmin(adminId);
    const device = await this.getDevice(request.device_id);
    if (device.status !== "online") {
      throw conflict("DEVICE_OFFLINE", "Device is offline", true);
    }
    const { kind, shell, command } = this.validateJobRequest(request, device.capabilities);
    const riskLevel = this.riskLevel(kind, command);
    const id = this.id("job");
    const payload: JobPayload = {
      shell,
      command,
      stdout_limit_bytes: request.stdout_limit_bytes,
      stderr_limit_bytes: request.stderr_limit_bytes,
      created_by: adminId
    };
    const result = await this.pool.query<JobRow>(
      `insert into jobs (id, session_id, device_id, created_by_admin_id, kind, risk_level, status, payload_json, timeout_sec)
       values ($1, $2, $3, $4, $5, $6, 'queued', $7, $8)
       returning *`,
      [id, request.session_id ?? null, request.device_id, adminId, kind, riskLevel, JSON.stringify(payload), request.timeout_sec]
    );
    const job = this.mapJob(this.expectRow(result.rows[0], "JOB_NOT_CREATED", "Job was not created"));
    await this.audit(adminId, job.device_id, job.session_id, job.id, "job.created", riskLevel, "success", {
      kind: job.kind,
      shell: job.shell
    });
    return job;
  }

  public async createRebootJob(deviceId: string, confirm: boolean, reason: string | undefined, adminId: string): Promise<Job> {
    if (!confirm) {
      throw forbidden("CONFIRMATION_REQUIRED", "Reboot requires explicit confirmation");
    }
    const job = await this.createJob(
      {
        device_id: deviceId,
        kind: "system.reboot",
        timeout_sec: 60,
        stdout_limit_bytes: 65_536,
        stderr_limit_bytes: 65_536,
        confirm: true,
        ...(reason === undefined ? {} : { reason })
      },
      adminId
    );
    await this.audit(adminId, deviceId, null, job.id, "reboot.requested", "medium", "success", { reason: reason ?? null });
    return job;
  }

  public async markJobDispatched(jobId: string): Promise<JobForAgent> {
    const result = await this.pool.query<JobRow>("update jobs set status = 'dispatched', dispatched_at = now() where id = $1 returning *", [jobId]);
    const job = this.mapJob(this.expectRow(result.rows[0], "JOB_NOT_FOUND", "Job was not found"));
    await this.audit(job.created_by, job.device_id, job.session_id, job.id, "job.dispatched", job.risk_level, "success", {});
    return jobToAgentJob(job);
  }

  public async markJobRunning(jobId: string): Promise<void> {
    await this.pool.query("update jobs set status = 'running', started_at = now() where id = $1", [jobId]);
  }

  public async completeJob(result: JobResult): Promise<void> {
    const job = await this.getJob(result.job_id);
    const status = result.status === "timed_out" ? "timed_out" : result.status;
    await this.pool.query("begin");
    try {
      await this.pool.query("update jobs set status = $2, started_at = $3, finished_at = $4 where id = $1", [
        job.id,
        status,
        result.started_at,
        result.finished_at
      ]);
      await this.pool.query(
        `insert into job_results (
          id, job_id, device_id, status, exit_code, stdout, stderr, stdout_truncated, stderr_truncated,
          error_code, error_message, started_at, finished_at, duration_ms
        ) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
        on conflict (job_id) do update set
          status = excluded.status,
          exit_code = excluded.exit_code,
          stdout = excluded.stdout,
          stderr = excluded.stderr,
          stdout_truncated = excluded.stdout_truncated,
          stderr_truncated = excluded.stderr_truncated,
          error_code = excluded.error_code,
          error_message = excluded.error_message,
          started_at = excluded.started_at,
          finished_at = excluded.finished_at,
          duration_ms = excluded.duration_ms`,
        [
          this.id("res"),
          result.job_id,
          result.device_id,
          result.status,
          result.exit_code,
          result.stdout,
          result.stderr,
          result.stdout_truncated,
          result.stderr_truncated,
          result.error_code,
          result.error_message,
          result.started_at,
          result.finished_at,
          result.duration_ms
        ]
      );
      await this.pool.query("commit");
    } catch (error) {
      await this.pool.query("rollback");
      throw error;
    }
    await this.audit(job.created_by, job.device_id, job.session_id, job.id, "job.completed", job.risk_level, result.status, {
      exit_code: result.exit_code,
      error_code: result.error_code,
      duration_ms: result.duration_ms
    });
  }

  public async getJob(jobId: string): Promise<Job> {
    const result = await this.pool.query<JobRow>("select * from jobs where id = $1", [jobId]);
    const row = result.rows[0];
    if (row === undefined) {
      throw notFound("JOB_NOT_FOUND", "Job was not found");
    }
    return this.mapJob(row);
  }

  public async getJobResult(jobId: string): Promise<JobResult> {
    const result = await this.pool.query<JobResultRow>("select * from job_results where job_id = $1", [jobId]);
    const row = result.rows[0];
    if (row === undefined) {
      throw notFound("JOB_RESULT_NOT_FOUND", "Job result was not found");
    }
    return this.mapJobResult(row);
  }

  public async listAudit(): Promise<AuditEvent[]> {
    const result = await this.pool.query<AuditRow>("select * from audit_events order by created_at desc limit 500");
    return result.rows.map((row) => this.mapAudit(row));
  }

  public async queuedJobsForDevice(deviceId: string): Promise<Job[]> {
    const result = await this.pool.query<JobRow>("select * from jobs where device_id = $1 and status = 'queued' order by created_at asc", [deviceId]);
    return result.rows.map((row) => this.mapJob(row));
  }

  public async snapshot(): Promise<StoreSnapshot> {
    return {
      devices: await this.listDevices("admin_mvp"),
      jobs: (await this.pool.query<JobRow>("select * from jobs")).rows.map((row) => this.mapJob(row)),
      sessions: (await this.pool.query<SessionRow>("select id, device_id, name, status, created_at, closed_at from sessions")).rows.map((row) =>
        this.mapSession(row)
      ),
      auditEvents: await this.listAudit()
    };
  }

  private async ensureAdmin(adminId: string): Promise<void> {
    await this.pool.query(
      `insert into administrators (id, name, status)
       values ($1, $2, 'active')
       on conflict (id) do nothing`,
      [adminId, adminId]
    );
  }

  private async getDeviceRow(deviceId: string): Promise<DeviceRow> {
    const result = await this.pool.query<DeviceRow>("select * from devices where id = $1", [deviceId]);
    const row = result.rows[0];
    if (row === undefined) {
      throw notFound("DEVICE_NOT_FOUND", "Device was not found");
    }
    return row;
  }

  private validateJobRequest(request: JobCreateRequest, capabilities: Capability[]): { kind: JobKind; shell: Job["shell"]; command: string | null } {
    if (request.kind === "shell.exec") {
      if (request.shell === undefined || request.command === undefined) {
        throw badRequest("INVALID_JOB_REQUEST", "shell.exec requires shell and command");
      }
      const requiredCapability = capabilityForShell(request.shell);
      if (!capabilities.includes(requiredCapability)) {
        throw forbidden("UNSUPPORTED_CAPABILITY", `Device does not support ${requiredCapability}`);
      }
      return { kind: request.kind, shell: request.shell, command: request.command };
    }
    if (!capabilities.includes("reboot")) {
      throw forbidden("UNSUPPORTED_CAPABILITY", "Device does not support reboot");
    }
    if (!request.confirm) {
      throw forbidden("CONFIRMATION_REQUIRED", "system.reboot requires explicit confirmation");
    }
    return { kind: request.kind, shell: null, command: request.reason ?? "reboot" };
  }

  private riskLevel(kind: JobKind, command: string | undefined | null): RiskLevel {
    if (kind === "system.reboot") {
      return "medium";
    }
    const normalized = command?.toLowerCase() ?? "";
    if (
      normalized.includes("restart-service") ||
      normalized.includes("remove-item") ||
      normalized.includes("stop-process") ||
      normalized.includes("systemctl restart") ||
      normalized.includes(" rm ")
    ) {
      return "medium";
    }
    return "low";
  }

  private async audit(
    adminId: string | null,
    deviceId: string | null,
    sessionId: string | null,
    jobId: string | null,
    action: string,
    riskLevel: RiskLevel | null,
    result: string,
    metadata: Record<string, unknown>
  ): Promise<void> {
    if (adminId !== null) {
      await this.ensureAdmin(adminId);
    }
    await this.pool.query(
      `insert into audit_events (id, admin_id, device_id, session_id, job_id, action, risk_level, result, metadata_json)
       values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
      [this.id("aud"), adminId, deviceId, sessionId, jobId, action, riskLevel, result, JSON.stringify(metadata)]
    );
  }

  private mapDevice(row: DeviceRow): Device {
    return {
      id: row.id,
      name: row.name,
      status: row.status,
      os: {
        type: row.os_type,
        version: row.os_version,
        ...(row.distro === null ? {} : { distro: row.distro })
      },
      agent_version: row.agent_version,
      protocol_version: row.protocol_version,
      capabilities: row.capabilities_json,
      last_seen_at: row.last_seen_at === null ? null : row.last_seen_at.toISOString()
    };
  }

  private mapSession(row: SessionRow): Session {
    return {
      id: row.id,
      device_id: row.device_id,
      name: row.name,
      status: row.status,
      created_at: row.created_at.toISOString(),
      closed_at: row.closed_at === null ? null : row.closed_at.toISOString()
    };
  }

  private mapJob(row: JobRow): Job {
    return {
      id: row.id,
      device_id: row.device_id,
      session_id: row.session_id,
      kind: row.kind,
      shell: row.payload_json.shell,
      command: row.payload_json.command,
      timeout_sec: row.timeout_sec,
      stdout_limit_bytes: row.payload_json.stdout_limit_bytes,
      stderr_limit_bytes: row.payload_json.stderr_limit_bytes,
      status: row.status,
      risk_level: row.risk_level,
      created_by: row.created_by_admin_id,
      created_at: row.created_at.toISOString(),
      dispatched_at: row.dispatched_at === null ? null : row.dispatched_at.toISOString(),
      started_at: row.started_at === null ? null : row.started_at.toISOString(),
      finished_at: row.finished_at === null ? null : row.finished_at.toISOString()
    };
  }

  private mapJobResult(row: JobResultRow): JobResult {
    return {
      job_id: row.job_id,
      device_id: row.device_id,
      status: row.status,
      exit_code: row.exit_code,
      stdout: row.stdout,
      stderr: row.stderr,
      stdout_truncated: row.stdout_truncated,
      stderr_truncated: row.stderr_truncated,
      error_code: row.error_code,
      error_message: row.error_message,
      started_at: row.started_at.toISOString(),
      finished_at: row.finished_at.toISOString(),
      duration_ms: row.duration_ms
    };
  }

  private mapAudit(row: AuditRow): AuditEvent {
    return {
      id: row.id,
      admin_id: row.admin_id,
      device_id: row.device_id,
      session_id: row.session_id,
      job_id: row.job_id,
      action: row.action,
      risk_level: row.risk_level,
      result: row.result,
      metadata: row.metadata_json,
      created_at: row.created_at.toISOString()
    };
  }

  private expectRow<T>(row: T | undefined, code: string, message: string): T {
    if (row === undefined) {
      throw notFound(code, message);
    }
    return row;
  }

  private id(prefix: string): string {
    return `${prefix}_${randomUUID()}`;
  }
}
