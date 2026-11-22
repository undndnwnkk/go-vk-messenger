import type { ErrorResponse } from "./types";

export const NETWORK_ERROR = "network_error";
export const UNKNOWN_ERROR = "unknown_error";

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string, options?: ErrorOptions) {
    super(message, options);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export function isApiError(value: unknown): value is ApiError {
  return value instanceof ApiError;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export function isErrorResponse(value: unknown): value is ErrorResponse {
  if (!isRecord(value) || !isRecord(value.error)) {
    return false;
  }
  return typeof value.error.code === "string" && typeof value.error.message === "string";
}

export function toApiError(status: number, body: unknown): ApiError {
  if (isErrorResponse(body)) {
    return new ApiError(status, body.error.code, body.error.message);
  }
  return new ApiError(status, UNKNOWN_ERROR, `Request failed (${status})`);
}

const friendlyMessages: Record<string, string> = {
  [NETWORK_ERROR]: "Can't reach the server. Check your connection and try again.",
  invalid_credentials: "Wrong username or password.",
  user_exists: "That username is taken.",
  invalid_username: "Use 3 to 32 lowercase letters, digits or underscores.",
  invalid_password: "Use 8 to 72 characters.",
  rate_limited: "You're sending messages too fast. Wait a moment.",
  chat_not_found: "This chat doesn't exist or you're not a member.",
  service_unavailable: "The service is temporarily unavailable. Try again shortly.",
  internal_error: "Something went wrong on our side. Try again.",
};

export function errorMessage(error: unknown): string {
  if (isApiError(error)) {
    return friendlyMessages[error.code] ?? error.message;
  }
  return "Something went wrong. Try again.";
}

export function errorStatusLabel(error: unknown): string | null {
  if (!isApiError(error)) {
    return null;
  }
  return error.status === 0 ? "Network error" : `Request failed (${error.status})`;
}
