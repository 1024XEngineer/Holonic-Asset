import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  assetDetail: vi.fn(),
  detail: vi.fn(),
  record: vi.fn(),
  resolve: vi.fn(),
}));

vi.mock("./core-generation.api", () => ({
  coreGenerationApi: {
    detail: mocks.detail,
    resolveApplication: mocks.resolve,
  },
}));
vi.mock("../../asset/library/core-asset.api", () => ({
  coreAssetApi: { detail: mocks.assetDetail, record: mocks.record },
}));

import { resolveGenerationApplication } from "./generation-application.mutation";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.record.mockResolvedValue({ version: 5 });
  mocks.assetDetail.mockResolvedValue({
    content: {
      directionCount: 4,
      prototype: [{ id: 1, url: "hero.png" }],
      animations: [{ id: 2, name: "Idle", frames: [] }],
    },
  });
  mocks.resolve.mockResolvedValue({ completed: true });
});

describe("resolveGenerationApplication", () => {
  it("saves an applicable candidate before completing its task", async () => {
    mocks.detail.mockResolvedValue({
      status: "awaiting_application",
      result: {
        asset_id: 9,
        version: 4,
        content: {
          animations: [
            { id: 3, name: "Walk", frames: [{ id: 1, url: "walk.png" }] },
          ],
        },
      },
    });

    await resolveGenerationApplication({
      projectId: "7",
      assetId: "9",
      runId: "12",
      applied: true,
    });

    expect(mocks.record).toHaveBeenCalledWith({
      assetId: 9,
      expectedVersion: 4,
      content: {
        directionCount: 4,
        prototype: [{ id: 1, url: "hero.png" }],
        animations: [
          { id: 2, name: "Idle", frames: [] },
          {
            id: 3,
            name: "Walk",
            frames: [{ id: 1, url: "walk.png" }],
          },
        ],
      },
    });
    expect(mocks.resolve).toHaveBeenCalledWith(12, true);
    expect(mocks.record.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.resolve.mock.invocationCallOrder[0]!,
    );
  });

  it("discards without writing asset data", async () => {
    await resolveGenerationApplication({
      projectId: "7",
      assetId: "9",
      runId: "12",
      applied: false,
    });

    expect(mocks.detail).not.toHaveBeenCalled();
    expect(mocks.assetDetail).not.toHaveBeenCalled();
    expect(mocks.record).not.toHaveBeenCalled();
    expect(mocks.resolve).toHaveBeenCalledWith(12, false);
  });

  it("keeps the task awaiting when the candidate cannot be saved", async () => {
    mocks.detail.mockResolvedValue({
      status: "awaiting_application",
      result: { asset_id: 9, version: 4, content: {} },
    });
    mocks.record.mockRejectedValue(new Error("version conflict"));

    await expect(
      resolveGenerationApplication({
        projectId: "7",
        assetId: "9",
        runId: "12",
        applied: true,
      }),
    ).rejects.toThrow("version conflict");
    expect(mocks.resolve).not.toHaveBeenCalled();
  });
});
