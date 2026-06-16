import type { ApiEnvelope, CliConfig } from "./types.js";
import { CliError } from "./types.js";

export class ApiClient {
  private readonly config: CliConfig;

  public constructor(config: CliConfig) {
    this.config = config;
  }

  public async get<T>(path: string): Promise<T> {
    return this.request<T>("GET", path);
  }

  public async post<T>(path: string, body: unknown): Promise<T> {
    return this.request<T>("POST", path, body);
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const init: RequestInit = {
      method,
      headers: {
        authorization: `Bearer ${this.config.adminToken}`,
        "content-type": "application/json"
      }
    };
    if (body !== undefined) {
      init.body = JSON.stringify(body);
    }
    const response = await fetch(`${this.config.serverUrl}${path}`, init);

    const payload = (await response.json()) as ApiEnvelope<T>;
    if (!payload.ok) {
      throw new CliError(
        payload.error.code,
        payload.error.message,
        exitCodeForError(payload.error.code, response.status),
        payload.error.retryable,
        payload.error.details
      );
    }
    return payload.data;
  }
}

export function exitCodeForError(code: string, status: number): number {
  if (status === 401) {
    return 3;
  }
  if (status === 403) {
    if (code === "CONFIRMATION_REQUIRED") {
      return 8;
    }
    return 4;
  }
  if (code === "DEVICE_OFFLINE") {
    return 5;
  }
  if (code === "COMMAND_TIMED_OUT" || code === "JOB_RESULT_NOT_FOUND") {
    return 7;
  }
  if (code === "INCOMPATIBLE_PROTOCOL") {
    return 9;
  }
  if (status === 400) {
    return 2;
  }
  return 1;
}
