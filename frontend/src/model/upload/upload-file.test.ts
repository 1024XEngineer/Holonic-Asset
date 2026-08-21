import { afterEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ createTarget: vi.fn() }));

vi.mock("./upload.api", () => ({
  uploadApi: { createTarget: mocks.createTarget },
}));

import { uploadFile } from "./upload-file";

afterEach(() => {
  vi.clearAllMocks();
  vi.unstubAllGlobals();
});

describe("uploadFile", () => {
  it("requests an exact upload target and sends the Qiniu multipart form", async () => {
    const target = {
      objectKey: "uploads/reference.png",
      objectURL: "https://cdn.example/reference.png?token=signed",
      uploadURL: "https://upload.qiniup.com",
      uploadToken: "signed-policy",
    };
    const file = new File(["image"], "reference.png", { type: "image/png" });
    const controller = new AbortController();
    const fetchMock = vi.fn().mockResolvedValue(new Response(null));
    mocks.createTarget.mockResolvedValue(target);
    vi.stubGlobal("fetch", fetchMock);

    await expect(uploadFile(file, controller.signal)).resolves.toBe(target);

    expect(mocks.createTarget).toHaveBeenCalledWith(
      { contentType: "image/png", contentLength: 5 },
      controller.signal,
    );
    expect(fetchMock).toHaveBeenCalledWith(
      target.uploadURL,
      expect.objectContaining({
        method: "POST",
        signal: controller.signal,
      }),
    );
    const body = fetchMock.mock.calls[0]?.[1]?.body as FormData;
    expect(body.get("token")).toBe(target.uploadToken);
    expect(body.get("key")).toBe(target.objectKey);
    expect(body.get("file")).toEqual(file);
  });

  it("rejects invalid targets and failed storage responses", async () => {
    const file = new File(["image"], "reference.png", { type: "image/png" });
    mocks.createTarget.mockResolvedValueOnce({});
    await expect(uploadFile(file)).rejects.toThrow("invalid upload target");

    mocks.createTarget.mockResolvedValueOnce({
      objectKey: "uploads/reference.png",
      objectURL: "https://cdn.example/reference.png",
      uploadURL: "https://upload.qiniup.com",
      uploadToken: "signed-policy",
    });
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(null, { status: 500 })),
    );
    await expect(uploadFile(file)).rejects.toThrow("File upload failed (500)");
  });
});
