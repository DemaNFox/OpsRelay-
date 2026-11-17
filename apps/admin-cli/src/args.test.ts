import { describe, expect, it, vi } from "vitest";
import { parseCli } from "./args.js";
import { executeCommand } from "./commands.js";
import { renderError } from "./output.js";
import { CliError } from "./types.js";

describe("CLI argument parsing", () => {
  it("parses shell command options", () => {
    const parsed = parseCli(["ps", "dev_1", "Get-Service Spooler", "--json", "--save", "--session", "ses_1", "--timeout", "20"]);
    expect(parsed.command).toEqual(["ps", "dev_1", "Get-Service Spooler"]);
    expect(parsed.options.json).toBe(true);
    expect(parsed.options.save).toBe(true);
    expect(parsed.options.sessionId).toBe("ses_1");
    expect(parsed.options.timeoutSec).toBe(20);
  });

  it("parses the short enrollment token command", () => {
    const parsed = parseCli(["token", "client-pc-01", "--json"]);
    expect(parsed.command).toEqual(["token", "client-pc-01"]);
    expect(parsed.options.json).toBe(true);
  });

  it("creates a one-time token through the authenticated server API", async () => {
    const request = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(new Response(JSON.stringify({ ok: true, data: { token: "once" } }), { status: 200 }));
    try {
      const parsed = parseCli(["token", "client-pc-01", "--json"]);
      const result = await executeCommand(parsed.command, parsed.options, {
        serverUrl: "http://127.0.0.1:3030", adminToken: "test-admin-token-123", sessionDir: ".sessions"
      });
      expect(result).toEqual({ token: "once" });
      expect(request).toHaveBeenCalledWith("http://127.0.0.1:3030/v1/enrollment-tokens", expect.objectContaining({
        method: "POST", body: JSON.stringify({ name: "client-pc-01" }),
        headers: expect.objectContaining({ authorization: "Bearer test-admin-token-123" })
      }));
    } finally {
      request.mockRestore();
    }
  });

  it("renders machine-readable errors", () => {
    const rendered = renderError(new CliError("DEVICE_OFFLINE", "Device is offline", 5, true), true);
    expect(rendered.exitCode).toBe(5);
    expect(JSON.parse(rendered.output)).toEqual({
      ok: false,
      error: {
        code: "DEVICE_OFFLINE",
        message: "Device is offline",
        retryable: true
      }
    });
  });
});
