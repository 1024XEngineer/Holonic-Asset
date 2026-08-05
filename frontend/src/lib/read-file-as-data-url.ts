export function readFileAsDataUrl(
  file: File,
  signal?: AbortSignal,
): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    let settled = false;

    const cleanup = () => {
      signal?.removeEventListener("abort", abort);
    };

    const rejectRead = (error: unknown) => {
      if (settled) return;
      settled = true;
      cleanup();
      reject(
        error instanceof Error ? error : new Error("Unable to read file."),
      );
    };

    const abort = () => {
      if (reader.readyState === FileReader.LOADING) reader.abort();
      rejectRead(signal?.reason ?? createAbortError());
    };

    reader.addEventListener("load", () => {
      if (typeof reader.result !== "string") {
        rejectRead(new Error("Unable to read file as a data URL."));
        return;
      }
      settled = true;
      cleanup();
      resolve(reader.result);
    });
    reader.addEventListener("error", () =>
      rejectRead(reader.error ?? new Error("Unable to read file.")),
    );
    reader.addEventListener("abort", () =>
      rejectRead(signal?.reason ?? createAbortError()),
    );

    if (signal?.aborted) {
      rejectRead(signal.reason ?? createAbortError());
      return;
    }

    signal?.addEventListener("abort", abort, { once: true });

    try {
      reader.readAsDataURL(file);
    } catch (error) {
      rejectRead(error);
    }
  });
}

function createAbortError() {
  return new DOMException("File read aborted.", "AbortError");
}
