import { z } from "zod";

export const PROTOCOL_VERSION = "0.1.0";

export const osTypeSchema = z.enum(["windows", "linux"]);
export type OsType = z.infer<typeof osTypeSchema>;

export const deviceStatusSchema = z.enum(["offline", "online", "rebooting", "revoked", "unknown"]);
export type DeviceStatus = z.infer<typeof deviceStatusSchema>;

export const shellSchema = z.enum(["powershell", "cmd", "sh", "bash"]);
export type Shell = z.infer<typeof shellSchema>;

export const jobKindSchema = z.enum(["shell.exec", "system.reboot"]);
export type JobKind = z.infer<typeof jobKindSchema>;

export const jobStatusSchema = z.enum([
  "queued",
  "dispatched",
  "running",
  "completed",
  "failed",
  "timed_out",
  "expired",
  "cancelled",
  "interrupted_by_reboot"
]);
export type JobStatus = z.infer<typeof jobStatusSchema>;

export const riskLevelSchema = z.enum(["low", "medium", "high"]);
export type RiskLevel = z.infer<typeof riskLevelSchema>;

export const capabilitySchema = z.enum([
  "shell.powershell",
  "shell.cmd",
  "shell.sh",
  "shell.bash",
  "services.windows",
  "services.systemd",
  "logs.eventlog",
  "logs.journald",
  "files.read",
  "files.write",
  "reboot"
]);
export type Capability = z.infer<typeof capabilitySchema>;

export const osInfoSchema = z.object({
  type: osTypeSchema,
  version: z.string().min(1),
  distro: z.string().min(1).optional()
});
export type OsInfo = z.infer<typeof osInfoSchema>;

export const deviceSchema = z.object({
  id: z.string().min(1),
  name: z.string().min(1),
  status: deviceStatusSchema,
  os: osInfoSchema,
  agent_version: z.string().min(1),
  protocol_version: z.string().min(1),
  capabilities: z.array(capabilitySchema),
  last_seen_at: z.string().datetime().nullable()
});
export type Device = z.infer<typeof deviceSchema>;

export const apiErrorSchema = z.object({
  code: z.string().min(1),
  message: z.string().min(1),
  retryable: z.boolean(),
  details: z.record(z.unknown()).optional()
});
export type ApiError = z.infer<typeof apiErrorSchema>;

export const successEnvelopeSchema = <T extends z.ZodTypeAny>(data: T) =>
  z.object({ ok: z.literal(true), data });

export const errorEnvelopeSchema = z.object({
  ok: z.literal(false),
  error: apiErrorSchema
});

export const enrollmentTokenCreateRequestSchema = z.object({
  name: z.string().min(1).max(120),
  ttl_sec: z.number().int().positive().max(86_400).default(3600)
});
export type EnrollmentTokenCreateRequest = z.infer<typeof enrollmentTokenCreateRequestSchema>;

export const enrollmentTokenCreateResponseSchema = z.object({
  id: z.string().min(1),
  token: z.string().min(20),
  name: z.string().min(1),
  expires_at: z.string().datetime()
});
export type EnrollmentTokenCreateResponse = z.infer<typeof enrollmentTokenCreateResponseSchema>;

export const agentEnrollRequestSchema = z.object({
  token: z.string().min(20),
  name: z.string().min(1).max(120),
  os: osInfoSchema,
  agent_version: z.string().min(1),
  protocol_version: z.string().min(1),
  capabilities: z.array(capabilitySchema).min(1)
});
export type AgentEnrollRequest = z.infer<typeof agentEnrollRequestSchema>;

export const agentEnrollResponseSchema = z.object({
  device_id: z.string().min(1),
  device_secret: z.string().min(20),
  server_time: z.string().datetime(),
  protocol_version: z.string().min(1)
});
export type AgentEnrollResponse = z.infer<typeof agentEnrollResponseSchema>;

export const sessionCreateRequestSchema = z.object({
  device_id: z.string().min(1),
  name: z.string().min(1).max(120)
});
export type SessionCreateRequest = z.infer<typeof sessionCreateRequestSchema>;

export const sessionSchema = z.object({
  id: z.string().min(1),
  device_id: z.string().min(1),
  name: z.string().min(1),
  status: z.enum(["open", "closed", "failed"]),
  created_at: z.string().datetime(),
  closed_at: z.string().datetime().nullable()
});
export type Session = z.infer<typeof sessionSchema>;

export const jobCreateRequestSchema = z.object({
  device_id: z.string().min(1),
  session_id: z.string().min(1).optional(),
  kind: jobKindSchema,
  shell: shellSchema.optional(),
  command: z.string().min(1).max(16_384).optional(),
  timeout_sec: z.number().int().min(1).max(600).default(30),
  stdout_limit_bytes: z.number().int().min(1).max(4_194_304).default(1_048_576),
  stderr_limit_bytes: z.number().int().min(1).max(4_194_304).default(1_048_576),
  confirm: z.boolean().default(false),
  reason: z.string().max(500).optional()
});
export type JobCreateRequest = z.infer<typeof jobCreateRequestSchema>;

export const jobSchema = z.object({
  id: z.string().min(1),
  device_id: z.string().min(1),
  session_id: z.string().min(1).nullable(),
  kind: jobKindSchema,
  shell: shellSchema.nullable(),
  command: z.string().nullable(),
  timeout_sec: z.number().int().positive(),
  stdout_limit_bytes: z.number().int().positive(),
  stderr_limit_bytes: z.number().int().positive(),
  status: jobStatusSchema,
  risk_level: riskLevelSchema,
  created_by: z.string().min(1),
  created_at: z.string().datetime(),
  dispatched_at: z.string().datetime().nullable(),
  started_at: z.string().datetime().nullable(),
  finished_at: z.string().datetime().nullable()
});
export type Job = z.infer<typeof jobSchema>;

export const jobForAgentSchema = z.object({
  job_id: z.string().min(1),
  session_id: z.string().min(1).nullable(),
  device_id: z.string().min(1),
  kind: jobKindSchema,
  shell: shellSchema.nullable(),
  command: z.string().nullable(),
  timeout_sec: z.number().int().positive(),
  stdout_limit_bytes: z.number().int().positive(),
  stderr_limit_bytes: z.number().int().positive(),
  created_by: z.string().min(1)
});
export type JobForAgent = z.infer<typeof jobForAgentSchema>;

export const jobResultSchema = z.object({
  job_id: z.string().min(1),
  device_id: z.string().min(1),
  status: z.enum(["completed", "failed", "timed_out"]),
  exit_code: z.number().int().nullable(),
  stdout: z.string(),
  stderr: z.string(),
  stdout_truncated: z.boolean(),
  stderr_truncated: z.boolean(),
  error_code: z.string().min(1).nullable(),
  error_message: z.string().min(1).nullable(),
  started_at: z.string().datetime(),
  finished_at: z.string().datetime(),
  duration_ms: z.number().int().nonnegative()
});
export type JobResult = z.infer<typeof jobResultSchema>;

export const heartbeatSchema = z.object({
  type: z.literal("heartbeat"),
  device_id: z.string().min(1),
  agent_version: z.string().min(1),
  uptime_sec: z.number().int().nonnegative(),
  active_jobs: z.number().int().nonnegative(),
  sent_at: z.string().datetime()
});
export type Heartbeat = z.infer<typeof heartbeatSchema>;

export const agentHelloSchema = z.object({
  type: z.literal("hello"),
  protocol_version: z.string().min(1),
  agent_version: z.string().min(1),
  device_id: z.string().min(1),
  os: osInfoSchema,
  capabilities: z.array(capabilitySchema).min(1)
});
export type AgentHello = z.infer<typeof agentHelloSchema>;

export const serverJobMessageSchema = z.object({
  type: z.literal("job"),
  job: jobForAgentSchema
});
export type ServerJobMessage = z.infer<typeof serverJobMessageSchema>;

export const agentJobResultMessageSchema = z.object({
  type: z.literal("job_result"),
  result: jobResultSchema
});
export type AgentJobResultMessage = z.infer<typeof agentJobResultMessageSchema>;

export const agentMessageSchema = z.discriminatedUnion("type", [
  agentHelloSchema,
  heartbeatSchema,
  agentJobResultMessageSchema
]);
export type AgentMessage = z.infer<typeof agentMessageSchema>;

export function capabilityForShell(shell: Shell): Capability {
  return `shell.${shell}` as Capability;
}

export function jobToAgentJob(job: Job): JobForAgent {
  return {
    job_id: job.id,
    session_id: job.session_id,
    device_id: job.device_id,
    kind: job.kind,
    shell: job.shell,
    command: job.command,
    timeout_sec: job.timeout_sec,
    stdout_limit_bytes: job.stdout_limit_bytes,
    stderr_limit_bytes: job.stderr_limit_bytes,
    created_by: job.created_by
  };
}
