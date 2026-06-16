import type { ApiError } from "@opsrelay/protocol";

export type JsonValue = null | boolean | number | string | JsonValue[] | { [key: string]: JsonValue };

export interface CliConfig {
  serverUrl: string;
  adminToken: string;
  sessionDir: string;
}

export interface CliOptions {
  json: boolean;
  save: boolean;
  sessionId: string | null;
  timeoutSec: number | null;
  confirm: boolean;
  name: string | null;
  reason: string | null;
}

export interface ParsedCli {
  command: string[];
  options: CliOptions;
}

export interface SuccessEnvelope<T> {
  ok: true;
  data: T;
}

export interface ErrorEnvelope {
  ok: false;
  error: ApiError;
}

export type ApiEnvelope<T> = SuccessEnvelope<T> | ErrorEnvelope;

export class CliError extends Error {
  public readonly code: string;
  public readonly retryable: boolean;
  public readonly exitCode: number;
  public readonly details: Record<string, unknown> | undefined;

  public constructor(code: string, message: string, exitCode: number, retryable = false, details?: Record<string, unknown>) {
    super(message);
    this.code = code;
    this.retryable = retryable;
    this.exitCode = exitCode;
    this.details = details;
  }
}
