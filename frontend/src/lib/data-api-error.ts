export const dataApiErrorCodes = [
  "BAD_REQUEST",
  "NOT_FOUND",
  "CONFLICT",
  "UNAVAILABLE",
  "UNKNOWN",
] as const;

export type DataApiErrorCode = (typeof dataApiErrorCodes)[number];

export class DataApiError extends Error {
  readonly code: DataApiErrorCode;
  readonly details?: unknown;

  constructor(code: DataApiErrorCode, message: string, details?: unknown) {
    super(message);
    this.name = "DataApiError";
    this.code = code;
    this.details = details;
  }
}
