import { CliError } from "./types.js";

export function renderSuccess(data: unknown, asJson: boolean): string {
  if (asJson) {
    return `${JSON.stringify({ ok: true, data }, null, 2)}\n`;
  }
  return `${formatHuman(data)}\n`;
}

export function renderError(error: unknown, asJson: boolean): { output: string; exitCode: number } {
  const cliError =
    error instanceof CliError
      ? error
      : new CliError("UNEXPECTED_ERROR", error instanceof Error ? error.message : "Unexpected error", 1, false);
  const envelope = {
    ok: false,
    error: {
      code: cliError.code,
      message: cliError.message,
      retryable: cliError.retryable,
      ...(cliError.details === undefined ? {} : { details: cliError.details })
    }
  };
  return {
    output: asJson ? `${JSON.stringify(envelope, null, 2)}\n` : `${cliError.code}: ${cliError.message}\n`,
    exitCode: cliError.exitCode
  };
}

function formatHuman(data: unknown): string {
  if (Array.isArray(data)) {
    return data.map((item) => formatHuman(item)).join("\n");
  }
  if (typeof data === "object" && data !== null) {
    return JSON.stringify(data, null, 2);
  }
  return String(data);
}
