import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

class MockFileReader extends EventTarget {
  static readonly EMPTY = 0;
  static readonly LOADING = 1;
  static readonly DONE = 2;

  readyState = MockFileReader.EMPTY;
  result: string | ArrayBuffer | null = null;
  error: DOMException | null = null;

  readAsDataURL() {
    this.readyState = MockFileReader.LOADING;
    queueMicrotask(() => {
      if (this.readyState !== MockFileReader.LOADING) return;
      this.result = "data:image/png;base64,preview";
      this.readyState = MockFileReader.DONE;
      this.dispatchEvent(new Event("load"));
    });
  }

  abort() {
    if (this.readyState !== MockFileReader.LOADING) return;
    this.readyState = MockFileReader.DONE;
    this.dispatchEvent(new Event("abort"));
  }
}

const { readFileAsDataUrl } = await import("./read-file-as-data-url");

beforeEach(() => vi.stubGlobal("FileReader", MockFileReader));
afterEach(() => vi.unstubAllGlobals());

describe("readFileAsDataUrl", () => {
  it("resolves with the data URL from the reader", async () => {
    await expect(
      readFileAsDataUrl(new File(["image"], "preview.png")),
    ).resolves.toBe("data:image/png;base64,preview");
  });

  it("rejects when the read is aborted", async () => {
    const controller = new AbortController();
    const promise = readFileAsDataUrl(
      new File(["image"], "preview.png"),
      controller.signal,
    );

    controller.abort();

    await expect(promise).rejects.toMatchObject({ name: "AbortError" });
  });

  it("rejects when the reader reports an error", async () => {
    class ErrorFileReader extends MockFileReader {
      override readAsDataURL() {
        this.readyState = MockFileReader.LOADING;
        queueMicrotask(() => {
          this.error = new DOMException("Read failed", "NotReadableError");
          this.readyState = MockFileReader.DONE;
          this.dispatchEvent(new Event("error"));
        });
      }
    }

    vi.stubGlobal("FileReader", ErrorFileReader);

    await expect(
      readFileAsDataUrl(new File(["image"], "preview.png")),
    ).rejects.toMatchObject({ name: "NotReadableError" });
  });
});
