// @vitest-environment happy-dom

import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { AssetExportResponse } from "./export.contract";

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  get: vi.fn(),
}));

vi.mock("./export.api", () => ({
  coreExportApi: {
    create: mocks.create,
    get: mocks.get,
  },
}));

import { useAssetExport } from "./use-asset-export";

beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers();
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("useAssetExport", () => {
  it("creates, polls, and downloads a completed export", async () => {
    const result = completedExport();
    mocks.create.mockResolvedValue({ exportId: 12 });
    mocks.get.mockResolvedValueOnce({ ...result, status: "processing" });
    mocks.get.mockResolvedValue(result);
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click");
    const { result: hook } = renderHook(() => useAssetExport());

    const exportPromise = startExport(hook, 9, 2_000);
    await expect(exportPromise).resolves.toBeUndefined();

    expect(mocks.create).toHaveBeenCalledWith({ assetId: 9 });
    expect(mocks.get).toHaveBeenNthCalledWith(1, 12);
    expect(mocks.get).toHaveBeenNthCalledWith(2, 12);
    expect(click).toHaveBeenCalledOnce();
    expect(hook.current.state).toEqual({ phase: "completed", result });
    expect(hook.current.isExporting).toBe(false);
  });

  it("reports an export API failure", async () => {
    mocks.create.mockRejectedValue(new Error("queue unavailable"));
    const { result } = renderHook(() => useAssetExport());

    const exportPromise = startExport(result, 9, 0);
    await expect(exportPromise).rejects.toThrow("queue unavailable");
    expect(result.current.state).toEqual({
      phase: "failed",
      message: "queue unavailable",
    });
  });

  it("reports failed and cancelled exports with their server error", async () => {
    mocks.create.mockResolvedValue({ exportId: 12 });
    mocks.get.mockResolvedValue({
      ...completedExport(),
      status: "failed",
      error: "package failed",
    });
    const { result } = renderHook(() => useAssetExport());

    const failedPromise = startExport(result, 9, 1_000);
    await expect(failedPromise).rejects.toThrow("package failed");
    expect(result.current.state).toEqual({
      phase: "failed",
      message: "package failed",
    });

    mocks.get.mockResolvedValue({
      ...completedExport(),
      status: "cancelled",
    });
    const cancelledPromise = startExport(result, 9, 1_000);
    await expect(cancelledPromise).rejects.toThrow("Export failed.");
  });

  it("rejects completed exports without a download URL", async () => {
    mocks.create.mockResolvedValue({ exportId: 12 });
    mocks.get.mockResolvedValue({
      ...completedExport(),
      downloadUrl: undefined,
    });
    const { result } = renderHook(() => useAssetExport());

    const exportPromise = startExport(result, 9, 1_000);
    await expect(exportPromise).rejects.toThrow(
      "Export download is unavailable.",
    );
  });

  it("uses a fallback message for non-Error failures", async () => {
    mocks.create.mockRejectedValue("offline");
    const { result } = renderHook(() => useAssetExport());

    const exportPromise = startExport(result, 9, 0);
    await expect(exportPromise).rejects.toBe("offline");
    expect(result.current.state).toEqual({
      phase: "failed",
      message: "Export failed.",
    });
  });

  it("ignores a result from an operation superseded by a newer export", async () => {
    let resolveCreate!: (value: { exportId: number }) => void;
    const createPromise = new Promise<{ exportId: number }>((resolve) => {
      resolveCreate = resolve;
    });
    mocks.create.mockReturnValueOnce(createPromise).mockResolvedValue({
      exportId: 13,
    });
    mocks.get.mockResolvedValue({ ...completedExport(), exportId: 13 });
    const { result } = renderHook(() => useAssetExport());

    let firstPromise!: Promise<void>;
    await act(async () => {
      firstPromise = result.current.exportAsset(9);
      await Promise.resolve();
      const secondPromise = result.current.exportAsset(10);
      await vi.advanceTimersByTimeAsync(1_000);
      await secondPromise;
    });
    resolveCreate({ exportId: 12 });
    await expect(firstPromise).resolves.toBeUndefined();
    expect(mocks.get).toHaveBeenCalledWith(13);
  });
});

async function startExport(
  hook: { current: ReturnType<typeof useAssetExport> },
  assetId: number,
  elapsedMilliseconds: number,
) {
  let exportPromise!: Promise<void>;
  await act(async () => {
    exportPromise = hook.current.exportAsset(assetId);
    void exportPromise.catch(() => undefined);
    await Promise.resolve();
    await vi.advanceTimersByTimeAsync(elapsedMilliseconds);
  });
  return exportPromise;
}

function completedExport(): AssetExportResponse {
  return {
    assetId: 9,
    downloadUrl: "https://cdn.example.test/export.zip",
    exportId: 12,
    fileName: "forest.zip",
    fileSize: 128,
    recordId: 4,
    sha256: "hash",
    status: "completed",
    version: 2,
  };
}
