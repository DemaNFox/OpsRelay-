import type { ApiError } from "@opsrelay/protocol";

export class AppError extends Error {
  public readonly statusCode: number;
  public readonly apiError: ApiError;

  public constructor(statusCode: number, apiError: ApiError) {
    super(apiError.message);
    this.statusCode = statusCode;
    this.apiError = apiError;
  }
}

export function badRequest(code: string, message: string, details?: Record<string, unknown>): AppError {
  return new AppError(400, { code, message, retryable: false, details });
}

export function unauthorized(message = "Authentication required"): AppError {
  return new AppError(401, { code: "AUTHENTICATION_REQUIRED", message, retryable: false });
}

export function forbidden(code: string, message: string): AppError {
  return new AppError(403, { code, message, retryable: false });
}

export function notFound(code: string, message: string): AppError {
  return new AppError(404, { code, message, retryable: false });
}

export function conflict(code: string, message: string, retryable = false): AppError {
  return new AppError(409, { code, message, retryable });
}
