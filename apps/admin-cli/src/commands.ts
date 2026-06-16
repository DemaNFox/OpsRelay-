import type { Device, Job, JobResult, Session } from "@opsrelay/protocol";
import { ApiClient } from "./apiClient.js";
import { saveJobArtifact, saveSession } from "./sessionArtifacts.js";
import { CliError, type CliConfig, type CliOptions } from "./types.js";

export async function executeCommand(command: string[], options: CliOptions, config: CliConfig): Promise<unknown> {
  const api = new ApiClient(config);
  const [first, second, third, fourth] = command;

  if (first === "devices" && second === "list") {
    return api.get<Device[]>("/v1/devices");
  }
  if (first === "device" && second === "info" && third !== undefined) {
    return api.get<Device>(`/v1/devices/${encodeURIComponent(third)}`);
  }
  if (first === "enroll-token" && second === "create") {
    if (options.name === null) {
      throw new CliError("INVALID_ARGUMENTS", "enroll-token create requires --name", 2);
    }
    return api.post("/v1/enrollment-tokens", { name: options.name });
  }
  if (first === "session" && second === "start" && third !== undefined) {
    if (options.name === null) {
      throw new CliError("INVALID_ARGUMENTS", "session start requires --name", 2);
    }
    const session = await api.post<Session>("/v1/sessions", { device_id: third, name: options.name });
    await saveSession(session, config);
    return session;
  }
  if (first === "job" && second === "status" && third !== undefined) {
    return api.get<Job>(`/v1/jobs/${encodeURIComponent(third)}`);
  }
  if (first === "job" && second === "result" && third !== undefined) {
    const result = await api.get<JobResult>(`/v1/jobs/${encodeURIComponent(third)}/result`);
    if (options.save) {
      const job = await api.get<Job>(`/v1/jobs/${encodeURIComponent(third)}`);
      await saveJobArtifact(job, result, config);
    }
    return result;
  }
  if ((first === "ps" || first === "cmd" || first === "sh") && second !== undefined && third !== undefined) {
    return runShellJob(api, first, second, collectCommandText(command.slice(2)), options, config);
  }
  if (first === "reboot" && second !== undefined) {
    if (!options.confirm) {
      throw new CliError("CONFIRMATION_REQUIRED", "reboot requires --confirm", 8);
    }
    return api.post(`/v1/devices/${encodeURIComponent(second)}/reboot`, {
      confirm: true,
      ...(options.reason === null ? {} : { reason: options.reason })
    });
  }

  throw new CliError("INVALID_ARGUMENTS", usage(), 2);
}

async function runShellJob(
  api: ApiClient,
  mode: "ps" | "cmd" | "sh",
  deviceId: string,
  command: string,
  options: CliOptions,
  config: CliConfig
): Promise<JobResult> {
  const shell = mode === "ps" ? "powershell" : mode;
  const timeoutSec = options.timeoutSec ?? 30;
  const job = await api.post<Job>("/v1/jobs", {
    device_id: deviceId,
    session_id: options.sessionId ?? undefined,
    kind: "shell.exec",
    shell,
    command,
    timeout_sec: timeoutSec
  });

  const result = await waitForResult(api, job.id, timeoutSec + 15);
  if (options.save || options.sessionId !== null) {
    await saveJobArtifact(job, result, config);
  }
  if (result.status === "failed") {
    throw new CliError("JOB_FAILED", "Remote job failed", 6, false, { job_id: job.id, exit_code: result.exit_code });
  }
  if (result.status === "timed_out") {
    throw new CliError("COMMAND_TIMED_OUT", "Remote command timed out", 7, false, { job_id: job.id });
  }
  return result;
}

async function waitForResult(api: ApiClient, jobId: string, maxWaitSec: number): Promise<JobResult> {
  const deadline = Date.now() + maxWaitSec * 1000;
  let lastError: CliError | null = null;
  while (Date.now() < deadline) {
    try {
      return await api.get<JobResult>(`/v1/jobs/${encodeURIComponent(jobId)}/result`);
    } catch (error) {
      if (error instanceof CliError && error.code === "JOB_RESULT_NOT_FOUND") {
        lastError = error;
        await sleep(500);
        continue;
      }
      throw error;
    }
  }
  throw new CliError("COMMAND_TIMED_OUT", "Timed out waiting for remote job result", 7, false, { job_id: jobId, last_error: lastError?.message });
}

function collectCommandText(values: string[]): string {
  return values.join(" ");
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

export function usage(): string {
  return [
    "Usage:",
    "  opsrelay devices list [--json]",
    "  opsrelay device info <device-id> [--json]",
    "  opsrelay enroll-token create --name <name> [--json]",
    "  opsrelay session start <device-id> --name <name> [--json]",
    "  opsrelay ps|cmd|sh <device-id> <command> [--session <id>] [--timeout <sec>] [--json] [--save]",
    "  opsrelay job status <job-id> [--json]",
    "  opsrelay job result <job-id> [--json] [--save]",
    "  opsrelay reboot <device-id> --confirm [--reason <text>] [--json]"
  ].join("\n");
}
