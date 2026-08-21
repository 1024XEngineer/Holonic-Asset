import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  loadCore: vi.fn(),
  saveCore: vi.fn(),
}));

vi.mock("./core-asset-record", () => ({
  loadCoreAssetWorkspace: mocks.loadCore,
  saveCoreAssetRevision: mocks.saveCore,
}));

import { assetWorkspaceApi } from "./record.api";

const input = { projectId: "11", assetId: "9" };

beforeEach(() => vi.clearAllMocks());

describe("assetWorkspaceApi", () => {
  it("loads a persisted Core workspace", async () => {
    const workspace = { projectName: "Core project" };
    mocks.loadCore.mockResolvedValue(workspace);

    await expect(assetWorkspaceApi.load(input)).resolves.toBe(workspace);
    expect(mocks.loadCore).toHaveBeenCalledWith(input);
  });

  it("propagates Core loading failures", async () => {
    const error = new Error("Core unavailable");
    mocks.loadCore.mockRejectedValue(error);

    await expect(assetWorkspaceApi.load(input)).rejects.toBe(error);
  });

  it("saves persisted revisions through the Core adapter", async () => {
    const saved = { version: "v4" };
    const request = {
      ...input,
      record: { mode: "character", prompt: "Hero", character: {} },
    } as never;
    mocks.saveCore.mockResolvedValue(saved);

    await expect(assetWorkspaceApi.saveRevision(request)).resolves.toBe(saved);
    expect(mocks.saveCore).toHaveBeenCalledWith(request);
  });
});
