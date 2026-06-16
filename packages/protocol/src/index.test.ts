import { describe, expect, it } from "vitest";
import {
  agentEnrollRequestSchema,
  capabilityForShell,
  heartbeatSchema,
  jobCreateRequestSchema,
  jobResultSchema
} from "./index.js";

describe("protocol schemas", () => {
  it("accepts a valid heartbeat", () => {
    const result = heartbeatSchema.safeParse({
      type: "heartbeat",
      device_id: "dev_123",
      agent_version: "0.1.0",
      uptime_sec: 10,
      active_jobs: 0,
      sent_at: "2026-06-16T10:00:00.000Z"
    });

    expect(result.success).toBe(true);
  });

  it("rejects unsupported shells", () => {
    const result = jobCreateRequestSchema.safeParse({
      device_id: "dev_123",
      kind: "shell.exec",
      shell: "fish",
      command: "echo test"
    });

    expect(result.success).toBe(false);
  });

  it("accepts a structured job result", () => {
    const result = jobResultSchema.safeParse({
      job_id: "job_123",
      device_id: "dev_123",
      status: "completed",
      exit_code: 0,
      stdout: "ok",
      stderr: "",
      stdout_truncated: false,
      stderr_truncated: false,
      error_code: null,
      error_message: null,
      started_at: "2026-06-16T10:00:00.000Z",
      finished_at: "2026-06-16T10:00:01.000Z",
      duration_ms: 1000
    });

    expect(result.success).toBe(true);
  });

  it("requires enrollment capabilities", () => {
    const result = agentEnrollRequestSchema.safeParse({
      token: "enr_12345678901234567890",
      name: "office-pc-01",
      os: { type: "windows", version: "11" },
      agent_version: "0.1.0",
      protocol_version: "0.1.0",
      capabilities: []
    });

    expect(result.success).toBe(false);
  });

  it("maps shell to capability", () => {
    expect(capabilityForShell("powershell")).toBe("shell.powershell");
  });
});
