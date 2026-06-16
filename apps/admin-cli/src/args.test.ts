import { describe, expect, it } from "vitest";
import { parseCli } from "./args.js";
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
