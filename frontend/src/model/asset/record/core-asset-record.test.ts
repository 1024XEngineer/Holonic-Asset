import { beforeEach, describe, expect, it, vi } from "vitest";

import type {
  AssetDetailResponse,
  AssetRecordResponse,
} from "../library/asset.contract";

const mocks = vi.hoisted(() => ({
  assetDetail: vi.fn(),
  assetRecords: vi.fn(),
  assetRecord: vi.fn(),
  projectDetail: vi.fn(),
}));

vi.mock("../library/core-asset.api", () => ({
  coreAssetApi: {
    detail: mocks.assetDetail,
    records: mocks.assetRecords,
    record: mocks.assetRecord,
  },
}));
vi.mock("../../project", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../project")>()),
  projectApi: { detail: mocks.projectDetail },
}));

import {
  loadCoreAssetWorkspace,
  saveCoreAssetRevision,
  toCoreAudioAssetWorkspace,
  toCoreSceneryAssetWorkspace,
  toCoreTilesetAssetWorkspace,
  toCoreUISetAssetWorkspace,
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
    "rejects a non-persisted asset ID: %s",
    async (assetId) => {
      await expect(
        loadCoreAssetWorkspace({ projectId: "11", assetId }),
      ).rejects.toThrow("requires a persisted Core API asset");
      expect(mocks.assetDetail).not.toHaveBeenCalled();
    },
  );

  it("loads audio metadata through the Core adapter", async () => {
    mocks.assetDetail.mockResolvedValue(audioDetail());

    await expect(
      loadCoreAssetWorkspace({ projectId: "11", assetId: "9" }),
    ).resolves.toMatchObject({
      projectName: "Demo",
      asset: { id: "9", kind: "audio", version: "v1" },
      record: {
        mode: "audio",
        prompt: "A quiet forest ambience",
        audio: {},
      },
    });
  });
});

describe("saveCoreAssetRevision", () => {
  it("persists scenery content and reloads real revision history", async () => {
    mocks.assetRecord.mockResolvedValue({ version: 4 });
    mocks.assetRecords.mockResolvedValue({ records: [sceneryRecord()] });

    await expect(
      saveCoreAssetRevision({
        projectId: "11",
        assetId: "9",
        version: "v3",
        description: "Adjusted the sky",
        record: {
          mode: "scenery",
          prompt: "A layered moonlit forest",
          scenery: {
            dimensions: { width: 1920, height: 1080 },
            layers: [
              {
                id: "sky-draft",
                label: "Sky",
                detail: "Night sky",
                imageUrl: "sky.png",
                blendMode: "multiply",
                position: { x: 12, y: 24 },
                visible: true,
              },
            ],
          },
        },
      }),
    ).resolves.toMatchObject({ version: "v4" });

    expect(mocks.assetRecord).toHaveBeenCalledWith({
      assetId: 9,
      expectedVersion: 3,
      description: "Adjusted the sky",
      content: {
        layers: [
          {
            id: 1,
            name: "Sky",
            resource: "sky.png",
            position: { x: 12, y: 24 },
            visible: true,
            metadata: { detail: "Night sky", blendMode: "multiply" },
          },
        ],
      },
    });
  });

  it("round-trips UI component coordinates through Core content", async () => {
    const workspace = toCoreUISetAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail: uiSetDetail(),
      records: [],
    });
    mocks.assetRecord.mockResolvedValue({ version: 2 });

    await saveCoreAssetRevision({
      projectId: "11",
      assetId: "9",
      version: "v1",
      record: workspace.record,
    });

    expect(workspace.record).toMatchObject({
      mode: "uiset",
      uiset: {
        dimensions: { width: 1920, height: 1080 },
        components: [
          {
            kind: "button",
            bounds: { x: 10, y: 20, width: 25, height: 10 },
          },
        ],
      },
    });
    expect(mocks.assetRecord).toHaveBeenCalledWith({
      assetId: 9,
      expectedVersion: 1,
      content: {
        components: [
          {
            id: 7,
            name: "Start",
            size: { width: 480, height: 108 },
            position: { x: 192, y: 216 },
            metadata: { kind: "button" },
          },
        ],
      },
    });
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

describe("non-image Core asset workspaces", () => {
  it("maps UI Set components without mock defaults", () => {
    const workspace = toCoreUISetAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail: uiSetDetail(),
      records: [],
    });

    expect(workspace.record).toMatchObject({
      mode: "uiset",
      prompt: "Main menu controls",
      uiset: { components: [{ id: "7", label: "Start", kind: "button" }] },
    });
  });

  it("maps audio metadata without mock defaults", () => {
    const workspace = toCoreAudioAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail: audioDetail(),
      records: [],
    });

    expect(workspace.record).toEqual({
      mode: "audio",
      prompt: "A quiet forest ambience",
      audio: {},
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

function uiSetDetail(): Extract<AssetDetailResponse, { type: "uiset" }> {
  return {
    assetId: 9,
    projectId: 11,
    name: "Main Menu",
    description: "Main menu controls",
    type: "uiset",
    perspective: "Top-Down",
    dimensions: { width: 1920, height: 1080 },
    tags: [],
    version: 1,
    content: {
      components: [
        {
          id: 7,
          name: "Start",
          size: { width: 480, height: 108 },
          position: { x: 192, y: 216 },
          metadata: { kind: "button" },
        },
      ],
    },
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
