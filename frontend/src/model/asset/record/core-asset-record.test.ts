import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AssetDetailResponse } from "../library/asset.contract";

const mocks = vi.hoisted(() => ({
  assetDetail: vi.fn(),
  assetRecords: vi.fn(),
  projectDetail: vi.fn(),
}));

vi.mock("../library/core-asset.api", () => ({
  coreAssetApi: {
    detail: mocks.assetDetail,
    records: mocks.assetRecords,
  },
}));
vi.mock("../../project", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../project")>()),
  projectApi: { detail: mocks.projectDetail },
}));

import {
  loadCoreAssetWorkspace,
  toCoreTilesetAssetWorkspace,
} from "./core-asset-record";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.projectDetail.mockResolvedValue({ name: "Demo" });
  mocks.assetRecords.mockResolvedValue({ records: [] });
});

describe("loadCoreAssetWorkspace", () => {
  it("loads a persisted tileset without falling back to mock data", async () => {
    mocks.assetDetail.mockResolvedValue(tilesetDetail());

    await expect(
      loadCoreAssetWorkspace({ projectId: "11", assetId: "9" }),
    ).resolves.toMatchObject({
      projectName: "Demo",
      asset: { id: "9", projectId: "11", kind: "tileset", version: "v3" },
      record: {
        mode: "tileset",
        tileset: {
          gridSize: 16,
          items: [
            {
              id: "1:Grass edge",
              label: "Grass edge",
              tiles: [
                [2, 1],
                [3, 1],
              ],
              tileUrls: ["/grass-edge-1.png", "/grass-edge-2.png"],
            },
          ],
        },
      },
    });
    expect(mocks.assetDetail).toHaveBeenCalledWith(9);
    expect(mocks.projectDetail).toHaveBeenCalledWith("11");
    expect(mocks.assetRecords).toHaveBeenCalledWith(9);
  });

  it.each(["draft", "0", "1.5"])(
    "skips a non-persisted asset ID: %s",
    async (assetId) => {
      await expect(
        loadCoreAssetWorkspace({ projectId: "11", assetId }),
      ).resolves.toBeUndefined();
      expect(mocks.assetDetail).not.toHaveBeenCalled();
    },
  );
});

describe("toCoreTilesetAssetWorkspace", () => {
  it("keeps the generated tile URLs aligned with their occupied cells", () => {
    const workspace = toCoreTilesetAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail: tilesetDetail(),
      records: [],
    });

    expect(workspace.record).toMatchObject({
      mode: "tileset",
      tileset: {
        gridSize: 16,
        items: [
          {
            tiles: [
              [2, 1],
              [3, 1],
            ],
            tileUrls: ["/grass-edge-1.png", "/grass-edge-2.png"],
          },
        ],
      },
    });
  });
});

function tilesetDetail(): Extract<AssetDetailResponse, { type: "tileSet" }> {
  return {
    assetId: 9,
    projectId: 11,
    name: "Forest Terrain",
    description: "A generated forest terrain set",
    type: "tileSet",
    perspective: "Top-Down",
    dimensions: {
      tileSize: { width: 16, height: 16 },
      tileAmount: { columns: 16, rows: 16 },
    },
    tags: [],
    version: 3,
    content: {
      items: [
        {
          name: "Grass edge",
          tiles: [
            { url: "/grass-edge-1.png", position: { x: 2, y: 1 } },
            { url: "/grass-edge-2.png", position: { x: 3, y: 1 } },
          ],
        },
      ],
    },
  };
}
