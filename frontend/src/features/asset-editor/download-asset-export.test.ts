// @vitest-environment happy-dom

import { afterEach, describe, expect, it, vi } from "vitest";

import type { AssetExportResponse } from "@/model";

import { downloadAssetExport } from "./download-asset-export";

afterEach(() => {
  document.body.replaceChildren();
  vi.restoreAllMocks();
});

describe("downloadAssetExport", () => {
  it("starts a browser download with the export metadata", () => {
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click");

    downloadAssetExport(completedExport());

    expect(click).toHaveBeenCalledOnce();
    expect(document.body.childElementCount).toBe(0);
  });

  it("does nothing when the export has no download URL", () => {
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click");

    downloadAssetExport({ ...completedExport(), downloadUrl: undefined });

    expect(click).not.toHaveBeenCalled();
  });
});

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
