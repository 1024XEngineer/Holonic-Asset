import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  loadCore: vi.fn(),
  saveCore: vi.fn(),
  loadMock: vi.fn(),
  saveMock: vi.fn(),
}));

vi.mock("./core-asset-record", () => ({
  loadCoreAssetWorkspace: mocks.loadCore,
}));
vi.mock("./core-sprite-record", () => ({
  saveCoreSpriteAssetRevision: mocks.saveCore,
}));
vi.mock("./mock", () => ({
  getMockAssetRecord: mocks.loadMock,
  saveMockAssetRecordRevision: mocks.saveMock,
}));

import { assetWorkspaceApi } from "./record.api";

const input = { projectId: "11", assetId: "9" };

beforeEach(() => vi.clearAllMocks());

describe("assetWorkspaceApi", () => {
  it("returns a persisted Core workspace without loading mock data", async () => {
    const workspace = { projectName: "Core project" };
    mocks.loadCore.mockResolvedValue(workspace);

    await expect(assetWorkspaceApi.load(input)).resolves.toBe(workspace);
    expect(mocks.loadMock).not.toHaveBeenCalled();
  });

  it("falls back to mock workspaces for unsupported Core records", async () => {
    const workspace = { projectName: "Mock project" };
    mocks.loadCore.mockResolvedValue(undefined);
    mocks.loadMock.mockResolvedValue(workspace);

    await expect(assetWorkspaceApi.load(input)).resolves.toBe(workspace);
    expect(mocks.loadMock).toHaveBeenCalledWith(input);
  });

  it("saves persisted sprite revisions through the Core adapter", async () => {
    const saved = { version: "v4" };
    const request = {
      ...input,
      record: { mode: "character", prompt: "Hero", character: {} },
    } as never;
    mocks.saveCore.mockResolvedValue(saved);

    await expect(assetWorkspaceApi.saveRevision(request)).resolves.toBe(saved);
    expect(mocks.saveMock).not.toHaveBeenCalled();
  });
});
