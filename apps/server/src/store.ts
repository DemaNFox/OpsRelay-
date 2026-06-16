import { randomUUID } from "node:crypto";
import {
  PROTOCOL_VERSION,
  capabilityForShell,
  type AgentEnrollRequest,
  type Capability,
  type Device,
  type Job,
  type JobCreateRequest,
  type JobForAgent,
  type JobKind,
  type JobResult,
  type RiskLevel,
  type Session,
  jobToAgentJob
} from "@opsrelay/protocol";
import { badRequest, conflict, forbidden, notFound } from "./errors.js";
import { createSecret, sha256 } from "./crypto.js";

export interface EnrollmentTokenRecord {
  id: string;
  tokenHash: string;
  name: string;
  expiresAt: string;
  usedAt: string | null;
  usedByDeviceId: string | null;
  createdAt: string;
}

export interface DeviceIdentity {
  deviceId: string;
  secretHash: string;
}

export interface AuditEvent {
  id: string;
  admin_id: string | null;
  device_id: string | null;
  session_id: string | null;
  job_id: string | null;
  action: string;
  risk_level: RiskLevel | null;
  result: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface StoreSnapshot {
  devices: Device[];
  jobs: Job[];
  sessions: Session[];
  auditEvents: AuditEvent[];
}

export type Awaitable<T> = T | Promise<T>;

export interface Store {
  createEnrollmentToken(name: string, ttlSec: number, adminId: string): Awaitable<{ id: string; token: string; expires_at: string }>;
  enrollDevice(request: AgentEnrollRequest): Awaitable<{ device: Device; device_secret: string }>;
  authenticateDevice(deviceId: string, deviceSecret: string): Awaitable<Device>;
  markDeviceOnline(deviceId: string): Awaitable<Device>;
  markDeviceOffline(deviceId: string): Awaitable<void>;
  updateHeartbeat(deviceId: string, agentVersion: string): Awaitable<Device>;
  listDevices(adminId: string): Awaitable<Device[]>;
  getDevice(deviceId: string): Awaitable<Device>;
  createSession(deviceId: string, name: string, adminId: string): Awaitable<Session>;
  getSession(sessionId: string): Awaitable<Session>;
  createJob(request: JobCreateRequest, adminId: string): Awaitable<Job>;
  createRebootJob(deviceId: string, confirm: boolean, reason: string | undefined, adminId: string): Awaitable<Job>;
  markJobDispatched(jobId: string): Awaitable<JobForAgent>;
  markJobRunning(jobId: string): Awaitable<void>;
  completeJob(result: JobResult): Awaitable<void>;
  getJob(jobId: string): Awaitable<Job>;
  getJobResult(jobId: string): Awaitable<JobResult>;
  listAudit(): Awaitable<AuditEvent[]>;
  queuedJobsForDevice(deviceId: string): Awaitable<Job[]>;
  snapshot(): Awaitable<StoreSnapshot>;
}

export class InMemoryStore implements Store {
  private readonly enrollmentTokens = new Map<string, EnrollmentTokenRecord>();
  private readonly devices = new Map<string, Device>();
  private readonly deviceIdentities = new Map<string, DeviceIdentity>();
  private readonly sessions = new Map<string, Session>();
  private readonly jobs = new Map<string, Job>();
  private readonly results = new Map<string, JobResult>();
  private readonly auditEvents: AuditEvent[] = [];

  public createEnrollmentToken(name: string, ttlSec: number, adminId: string): { id: string; token: string; expires_at: string } {
    const id = this.id("enr");
    const token = createSecret("enr");
    const now = new Date();
    const expiresAt = new Date(now.getTime() + ttlSec * 1000).toISOString();

    this.enrollmentTokens.set(id, {
      id,
      tokenHash: sha256(token),
      name,
      expiresAt,
      usedAt: null,
      usedByDeviceId: null,
      createdAt: now.toISOString()
    });
    this.audit(adminId, null, null, null, "enrollment_token.created", "low", "success", { enrollment_token_id: id, name });

    return { id, token, expires_at: expiresAt };
  }

  public enrollDevice(request: AgentEnrollRequest): { device: Device; device_secret: string } {
    if (request.protocol_version !== PROTOCOL_VERSION) {
      throw badRequest("INCOMPATIBLE_PROTOCOL", "Agent protocol version is not supported", {
        expected: PROTOCOL_VERSION,
        actual: request.protocol_version
      });
    }

    const tokenHash = sha256(request.token);
    const tokenRecord = Array.from(this.enrollmentTokens.values()).find((record) => record.tokenHash === tokenHash);
    if (tokenRecord === undefined) {
      throw forbidden("INVALID_ENROLLMENT_TOKEN", "Enrollment token is invalid");
    }
    if (tokenRecord.usedAt !== null) {
      throw forbidden("ENROLLMENT_TOKEN_USED", "Enrollment token has already been used");
    }
    if (Date.parse(tokenRecord.expiresAt) <= Date.now()) {
      throw forbidden("ENROLLMENT_TOKEN_EXPIRED", "Enrollment token has expired");
    }

    const deviceId = this.id("dev");
    const secret = createSecret("devsec");
    const now = new Date().toISOString();
    const device: Device = {
      id: deviceId,
      name: request.name,
      status: "offline",
      os: request.os,
      agent_version: request.agent_version,
      protocol_version: request.protocol_version,
      capabilities: request.capabilities,
      last_seen_at: null
    };

    tokenRecord.usedAt = now;
    tokenRecord.usedByDeviceId = deviceId;
    this.devices.set(deviceId, device);
    this.deviceIdentities.set(deviceId, { deviceId, secretHash: sha256(secret) });
    this.audit(null, deviceId, null, null, "device.enrolled", "low", "success", {
      enrollment_token_id: tokenRecord.id,
      device_name: request.name
    });

    return { device, device_secret: secret };
  }

  public authenticateDevice(deviceId: string, deviceSecret: string): Device {
    const device = this.getDevice(deviceId);
    const identity = this.deviceIdentities.get(deviceId);
    if (identity === undefined || identity.secretHash !== sha256(deviceSecret)) {
      throw forbidden("INVALID_DEVICE_IDENTITY", "Device identity is invalid");
    }
    if (device.status === "revoked") {
      throw forbidden("DEVICE_REVOKED", "Device has been revoked");
    }
    return device;
  }

  public markDeviceOnline(deviceId: string): Device {
    const device = this.getDevice(deviceId);
    const updated: Device = { ...device, status: "online", last_seen_at: new Date().toISOString() };
    this.devices.set(deviceId, updated);
    this.audit(null, deviceId, null, null, "device.connected", "low", "success", {});
    return updated;
  }

  public markDeviceOffline(deviceId: string): void {
    const device = this.devices.get(deviceId);
    if (device === undefined || device.status === "revoked") {
      return;
    }
    this.devices.set(deviceId, { ...device, status: "offline", last_seen_at: new Date().toISOString() });
    this.audit(null, deviceId, null, null, "device.disconnected", "low", "success", {});
  }

  public updateHeartbeat(deviceId: string, agentVersion: string): Device {
    const device = this.getDevice(deviceId);
    const updated: Device = { ...device, agent_version: agentVersion, status: "online", last_seen_at: new Date().toISOString() };
    this.devices.set(deviceId, updated);
    return updated;
  }

  public listDevices(adminId: string): Device[] {
    this.audit(adminId, null, null, null, "devices.listed", "low", "success", {});
    return Array.from(this.devices.values());
  }

  public getDevice(deviceId: string): Device {
    const device = this.devices.get(deviceId);
    if (device === undefined) {
      throw notFound("DEVICE_NOT_FOUND", "Device was not found");
    }
    return device;
  }

  public createSession(deviceId: string, name: string, adminId: string): Session {
    this.getDevice(deviceId);
    const now = new Date().toISOString();
    const session: Session = {
      id: this.id("ses"),
      device_id: deviceId,
      name,
      status: "open",
      created_at: now,
      closed_at: null
    };
    this.sessions.set(session.id, session);
    this.audit(adminId, deviceId, session.id, null, "session.created", "low", "success", { name });
    return session;
  }

  public getSession(sessionId: string): Session {
    const session = this.sessions.get(sessionId);
    if (session === undefined) {
      throw notFound("SESSION_NOT_FOUND", "Session was not found");
    }
    return session;
  }

  public createJob(request: JobCreateRequest, adminId: string): Job {
    const device = this.getDevice(request.device_id);
    if (device.status !== "online") {
      throw conflict("DEVICE_OFFLINE", "Device is offline", true);
    }
    const { kind, shell, command } = this.validateJobRequest(request, device.capabilities);
    const now = new Date().toISOString();
    const riskLevel = this.riskLevel(kind, command);
    const job: Job = {
      id: this.id("job"),
      device_id: request.device_id,
      session_id: request.session_id ?? null,
      kind,
      shell,
      command,
      timeout_sec: request.timeout_sec,
      stdout_limit_bytes: request.stdout_limit_bytes,
      stderr_limit_bytes: request.stderr_limit_bytes,
      status: "queued",
      risk_level: riskLevel,
      created_by: adminId,
      created_at: now,
      dispatched_at: null,
      started_at: null,
      finished_at: null
    };

    this.jobs.set(job.id, job);
    this.audit(adminId, job.device_id, job.session_id, job.id, "job.created", riskLevel, "success", {
      kind: job.kind,
      shell: job.shell
    });
    return job;
  }

  public createRebootJob(deviceId: string, confirm: boolean, reason: string | undefined, adminId: string): Job {
    if (!confirm) {
      throw forbidden("CONFIRMATION_REQUIRED", "Reboot requires explicit confirmation");
    }
    const request: JobCreateRequest = {
      device_id: deviceId,
      kind: "system.reboot",
      timeout_sec: 60,
      stdout_limit_bytes: 65_536,
      stderr_limit_bytes: 65_536,
      confirm: true,
      ...(reason === undefined ? {} : { reason })
    };
    const job = this.createJob(request, adminId);
    this.audit(adminId, deviceId, null, job.id, "reboot.requested", "medium", "success", { reason: reason ?? null });
    return job;
  }

  public markJobDispatched(jobId: string): JobForAgent {
    const job = this.getJob(jobId);
    const updated: Job = { ...job, status: "dispatched", dispatched_at: new Date().toISOString() };
    this.jobs.set(jobId, updated);
    this.audit(job.created_by, job.device_id, job.session_id, job.id, "job.dispatched", job.risk_level, "success", {});
    return jobToAgentJob(updated);
  }

  public markJobRunning(jobId: string): void {
    const job = this.getJob(jobId);
    this.jobs.set(jobId, { ...job, status: "running", started_at: new Date().toISOString() });
  }

  public completeJob(result: JobResult): void {
    const job = this.getJob(result.job_id);
    const mappedStatus = result.status;
    const status = mappedStatus === "timed_out" ? "timed_out" : mappedStatus;
    const updated: Job = {
      ...job,
      status,
      started_at: result.started_at,
      finished_at: result.finished_at
    };
    this.jobs.set(job.id, updated);
    this.results.set(job.id, result);
    this.audit(job.created_by, job.device_id, job.session_id, job.id, "job.completed", job.risk_level, result.status, {
      exit_code: result.exit_code,
      error_code: result.error_code,
      duration_ms: result.duration_ms
    });
  }

  public getJob(jobId: string): Job {
    const job = this.jobs.get(jobId);
    if (job === undefined) {
      throw notFound("JOB_NOT_FOUND", "Job was not found");
    }
    return job;
  }

  public getJobResult(jobId: string): JobResult {
    const result = this.results.get(jobId);
    if (result === undefined) {
      throw notFound("JOB_RESULT_NOT_FOUND", "Job result was not found");
    }
    return result;
  }

  public listAudit(): AuditEvent[] {
    return [...this.auditEvents];
  }

  public queuedJobsForDevice(deviceId: string): Job[] {
    return Array.from(this.jobs.values()).filter((job) => job.device_id === deviceId && job.status === "queued");
  }

  public snapshot(): StoreSnapshot {
    return {
      devices: Array.from(this.devices.values()),
      jobs: Array.from(this.jobs.values()),
      sessions: Array.from(this.sessions.values()),
      auditEvents: [...this.auditEvents]
    };
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

  private audit(
    adminId: string | null,
    deviceId: string | null,
    sessionId: string | null,
    jobId: string | null,
    action: string,
    riskLevel: RiskLevel | null,
    result: string,
    metadata: Record<string, unknown>
  ): void {
    this.auditEvents.push({
      id: this.id("aud"),
      admin_id: adminId,
      device_id: deviceId,
      session_id: sessionId,
      job_id: jobId,
      action,
      risk_level: riskLevel,
      result,
      metadata,
      created_at: new Date().toISOString()
    });
  }

  private id(prefix: string): string {
    return `${prefix}_${randomUUID()}`;
  }
}
