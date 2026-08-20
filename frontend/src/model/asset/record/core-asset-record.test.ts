import { beforeEach, describe, expect, it, vi } from "vitest";

import type {
  AssetDetailResponse,
  AssetRecordResponse,
} from "../library/asset.contract";

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
  toCoreSceneryAssetWorkspace,
  toCoreTilesetAssetWorkspace,
} from "./core-asset-record";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.projectDetail.mockResolvedValue({ name: "Demo" });
  mocks.assetRecords.mockResolvedValue({ records: [] });
});

describe("loadCoreAssetWorkspace", () => {
  it("loads a persisted scenery asset without falling back to mock data", async () => {
    mocks.assetDetail.mockResolvedValue(sceneryDetail());

    await expect(
      loadCoreAssetWorkspace({ projectId: "11", assetId: "9" }),
    ).resolves.toMatchObject({
      projectName: "Demo",
      asset: { id: "9", projectId: "11", kind: "scenery", version: "v3" },
      record: {
        mode: "scenery",
        scenery: {
          layers: [
            {
              id: "1",
              label: "Sky",
              imageUrl: "https://cdn.example/sky.png",
              blendMode: "normal",
            },
          ],
        },
      },
    });
    expect(mocks.assetDetail).toHaveBeenCalledWith(9);
    expect(mocks.projectDetail).toHaveBeenCalledWith("11");
    expect(mocks.assetRecords).toHaveBeenCalledWith(9);
  });

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

  it("skips an asset type that the editor cannot load", async () => {
    mocks.assetDetail.mockResolvedValue(audioDetail());

    await expect(
      loadCoreAssetWorkspace({ projectId: "11", assetId: "9" }),
    ).resolves.toBeUndefined();
    expect(mocks.projectDetail).not.toHaveBeenCalled();
    expect(mocks.assetRecords).not.toHaveBeenCalled();
  });
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

describe("toCoreSceneryAssetWorkspace", () => {
  it("maps persisted layer resources to scenery canvas layers", () => {
    const workspace = toCoreSceneryAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail: sceneryDetail(),
      records: [],
    });

    expect(workspace.record).toMatchObject({
      mode: "scenery",
      scenery: {
        layers: [
          {
            id: "1",
            label: "Sky",
            detail: "Sky",
            imageUrl: "https://cdn.example/sky.png",
            blendMode: "normal",
          },
        ],
      },
    });
  });

  it("maps canvas dimensions, transforms, and revision history", () => {
    const detail = sceneryDetail();
    const layer = detail.content?.layers?.[0];
    if (!layer) throw new Error("Expected a scenery layer.");
    layer.transform = {
      scale: { x: 1.25, y: 0.75 },
      rotation: 12.4,
    };

    const workspace = toCoreSceneryAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail,
      records: [sceneryRecord()],
    });

    expect(workspace.record).toMatchObject({
      mode: "scenery",
      scenery: {
        dimensions: { width: 1920, height: 1080 },
        layers: [
          {
            transform: {
              scale: { x: 1.25, y: 0.75 },
              rotation: 12.4,
            },
            visible: true,
            opacity: 1,
            zIndex: 0,
          },
        ],
      },
    });
    expect(workspace.asset.history).toEqual([
      {
        id: "21",
        version: "v3",
        description: "Generated scenery",
        savedAt: "2026-08-19T08:00:00Z",
        status: "ready",
        isCurrent: true,
      },
    ]);
  });

  it.each([
    {},
    { scale: "wide", rotation: 0 },
    { scale: { x: Number.POSITIVE_INFINITY, y: 1 }, rotation: 0 },
    { scale: { x: 1, y: Number.NaN }, rotation: 0 },
    { scale: { x: 1, y: 1 }, rotation: "upright" },
  ])("ignores an invalid persisted transform: %j", (transform) => {
    const detail = sceneryDetail();
    const layer = detail.content?.layers?.[0];
    if (!layer) throw new Error("Expected a scenery layer.");
    layer.transform = transform;

    const workspace = toCoreSceneryAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail,
      records: [],
    });

    expect(workspace.record).toMatchObject({
      mode: "scenery",
      scenery: { layers: [{ transform: undefined }] },
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

function sceneryDetail(): Extract<AssetDetailResponse, { type: "scenery" }> {
  return {
    assetId: 9,
    projectId: 11,
    name: "Moonlit Forest",
    description: "A layered moonlit forest",
    type: "scenery",
    perspective: "Top-Down",
    dimensions: { width: 1920, height: 1080 },
    tags: [],
    version: 3,
    content: {
      layers: [
        {
          id: 1,
          name: "Sky",
          resource: "https://cdn.example/sky.png",
          position: { x: 0, y: 0 },
          visible: true,
          opacity: 1,
          zIndex: 0,
        },
      ],
    },
  };
}

function audioDetail(): Extract<AssetDetailResponse, { type: "audio" }> {
  return {
    assetId: 9,
    projectId: 11,
    name: "Forest ambience",
    description: "A quiet forest ambience",
    type: "audio",
    perspective: "Top-Down",
    dimensions: null,
    tags: [],
    version: 1,
    content: {},
  };
}

function sceneryRecord(): AssetRecordResponse<"scenery"> {
  return {
    assetId: 9,
    contentId: 31,
    recordId: 21,
    name: "Moonlit Forest",
    description: "Generated scenery",
    dimensions: { width: 1920, height: 1080 },
    perspective: "Top-Down",
    version: 3,
    createdAt: "2026-08-19T08:00:00Z",
    content: sceneryDetail().content,
  };
}
