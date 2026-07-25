import { DataApiError } from "../api-error";

export const DEFAULT_MOCK_DELAY_MS = 300;

export type MockRequestOptions = {
  delayMs?: number;
};

export async function runMockRequest<T>(
  operation: () => T | Promise<T>,
  options: MockRequestOptions = {},
): Promise<T> {
  const delayMs = options.delayMs ?? DEFAULT_MOCK_DELAY_MS;

  if (!Number.isFinite(delayMs) || delayMs < 0) {
    throw new DataApiError(
      "BAD_REQUEST",
      "Mock request delay must be a non-negative finite number.",
      { delayMs },
    );
  }

  if (delayMs > 0) {
    await new Promise<void>((resolve) => {
      globalThis.setTimeout(resolve, delayMs);
    });
  }

  try {
    return await operation();
  } catch (error) {
    if (error instanceof DataApiError) {
      throw error;
    }

    throw new DataApiError("UNKNOWN", "Mock request failed.", error);
  }
}
