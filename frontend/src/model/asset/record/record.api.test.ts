import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  loadCore: vi.fn(),
  loadMock: vi.fn(),
  saveMock: vi.fn(),
}));

vi.mock("./core-asset-record", () => ({
  loadCoreAssetWorkspace: mocks.loadCore,
}));
vi.mock("./mock", () => ({
  getMockAssetRecord: mocks.loadMock,
  saveMockAssetRecordRevision: mocks.saveMock,
}));

import { recordApi } from "./record.api";

const input = { projectId: "11", assetId: "9" };

beforeEach(() => vi.clearAllMocks());

describe("recordApi", () => {
  it("returns a persisted Core workspace without loading mock data", async () => {
    const workspace = { projectName: "Core project" };
    mocks.loadCore.mockResolvedValue(workspace);

    await expect(recordApi.get(input)).resolves.toBe(workspace);
    expect(mocks.loadMock).not.toHaveBeenCalled();
  });

  it("falls back to mock workspaces for unsupported Core records", async () => {
    const workspace = { projectName: "Mock project" };
    mocks.loadCore.mockResolvedValue(undefined);
    mocks.loadMock.mockResolvedValue(workspace);

    await expect(recordApi.get(input)).resolves.toBe(workspace);
    expect(mocks.loadMock).toHaveBeenCalledWith(input);
  });
});
