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

  it("rejects audio assets because they do not have editable records", async () => {
    mocks.assetDetail.mockResolvedValue(audioDetail());

    await expect(
      loadCoreAssetWorkspace({ projectId: "11", assetId: "9" }),
    ).rejects.toThrow("Audio assets do not have editable records.");
  });

  it.each(["character", "object"] as const)(
    "loads a persisted %s sprite asset through the Core adapter",
    async (type) => {
      mocks.assetDetail.mockResolvedValue(spriteDetail(type));

      await expect(
        loadCoreAssetWorkspace({ projectId: "11", assetId: "9" }),
      ).resolves.toMatchObject({
        asset: { kind: type },
        record: { mode: type },
      });
    },
  );

  it("loads a persisted UI Set through the Core adapter", async () => {
    mocks.assetDetail.mockResolvedValue(uiSetDetail());

    await expect(
      loadCoreAssetWorkspace({ projectId: "11", assetId: "9" }),
    ).resolves.toMatchObject({
      asset: { kind: "uiset" },
      record: { mode: "uiset", uiset: { components: [{ label: "Start" }] } },
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

  it("persists Tileset content with and without generated tile URLs", async () => {
    mocks.assetRecord.mockResolvedValue({ version: 2 });

    await saveCoreAssetRevision({
      projectId: "11",
      assetId: "9",
      record: {
        mode: "tileset",
        prompt: "Forest terrain",
        tileset: {
          gridSize: 8,
          items: [
            {
              id: "draft-grass",
              label: "Grass",
              tiles: [
                [0, 0],
                [1, 0],
              ],
              tileUrls: ["grass.png"],
            },
          ],
        },
      },
    });

    expect(mocks.assetRecord).toHaveBeenCalledWith({
      assetId: 9,
      content: {
        items: [
          {
            name: "Grass",
            tiles: [
              { url: "grass.png", position: { x: 0, y: 0 } },
              { position: { x: 1, y: 0 } },
            ],
          },
        ],
      },
    });
  });

  it("uses raw UI Set bounds when its canvas dimensions are unavailable", async () => {
    mocks.assetRecord.mockResolvedValue({ version: 2 });

    await saveCoreAssetRevision({
      projectId: "11",
      assetId: "9",
      record: {
        mode: "uiset",
        prompt: "Overlay",
        uiset: {
          components: [
            {
              id: "draft-component",
              label: "Overlay",
              kind: "label",
              bounds: { x: 5, y: 10, width: 50, height: 25 },
            },
          ],
        },
      },
    });

    expect(mocks.assetRecord).toHaveBeenCalledWith({
      assetId: 9,
      content: {
        components: [
          {
            id: 1,
            name: "Overlay",
            size: { width: 50, height: 25 },
            position: { x: 5, y: 10 },
            metadata: { kind: "label" },
          },
        ],
      },
    });
  });

  it("serializes omitted scenery display fields and optional transform", async () => {
    mocks.assetRecord.mockResolvedValue({ version: 2 });

    await saveCoreAssetRevision({
      projectId: "11",
      assetId: "9",
      record: {
        mode: "scenery",
        prompt: "Forest",
        scenery: {
          layers: [
            {
              id: "2",
              label: "Fog",
              detail: "Soft fog",
              imageUrl: "fog.png",
              blendMode: "normal",
              transform: { scale: { x: 1, y: 1 }, rotation: 0 },
              opacity: 0,
              zIndex: 0,
            },
          ],
        },
      },
    });

    expect(mocks.assetRecord).toHaveBeenCalledWith({
      assetId: 9,
      content: {
        layers: [
          {
            id: 2,
            name: "Fog",
            resource: "fog.png",
            position: { x: 0, y: 0 },
            transform: { scale: { x: 1, y: 1 }, rotation: 0 },
            opacity: 0,
            zIndex: 0,
            metadata: { detail: "Soft fog", blendMode: "normal" },
          },
        ],
      },
    });
  });

  it("persists character and object revisions through their Core payloads", async () => {
    mocks.assetRecord.mockResolvedValue({ version: 2 });
    const sprite = {
      prototype: {
        format: "png-sprite-sheet" as const,
        imageUrl: "sprite.png",
        frameWidth: 32,
        frameHeight: 32,
        columns: 1,
        rows: 1,
      },
      animations: [],
      nodePositions: {},
    };

    await saveCoreAssetRevision({
      projectId: "11",
      assetId: "9",
      record: { mode: "character", prompt: "Hero", character: sprite },
    });
    await saveCoreAssetRevision({
      projectId: "11",
      assetId: "9",
      record: { mode: "object", prompt: "Chest", object: sprite },
    });
    expect(mocks.assetRecord).toHaveBeenLastCalledWith({
      assetId: 9,
      content: {
        directionCount: 1,
        prototype: [{ id: 1, url: "sprite.png" }],
        animations: [],
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

  it("uses empty item collections when generated content is absent", () => {
    const itemWithoutTiles = tilesetDetail();
    itemWithoutTiles.content = { items: [{ name: "Empty" }] };
    const noContent = tilesetDetail();
    delete noContent.content;

    const itemWorkspace = toCoreTilesetAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail: itemWithoutTiles,
      records: [],
    });
    const emptyWorkspace = toCoreTilesetAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail: noContent,
      records: [],
    });

    expect(itemWorkspace.record).toMatchObject({
      mode: "tileset",
      tileset: { items: [{ label: "Empty", tiles: [], tileUrls: [] }] },
    });
    expect(emptyWorkspace.record).toMatchObject({
      mode: "tileset",
      tileset: { items: [] },
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

  it("preserves persisted layer details and defaults omitted display properties", () => {
    const detail = sceneryDetail();
    const layer = detail.content?.layers?.[0];
    if (!layer) throw new Error("Expected a scenery layer.");
    layer.metadata = { detail: "Darkened sky", blendMode: "multiply" };
    delete layer.visible;
    delete layer.opacity;
    delete layer.zIndex;

    const workspace = toCoreSceneryAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail,
      records: [],
    });

    expect(workspace.record).toMatchObject({
      mode: "scenery",
      scenery: {
        layers: [
          {
            detail: "Darkened sky",
            blendMode: "multiply",
          },
        ],
      },
    });
    if (workspace.record.mode !== "scenery") {
      throw new Error("Expected a scenery workspace.");
    }
    expect(workspace.record.scenery.layers[0]).not.toHaveProperty("visible");
    expect(workspace.record.scenery.layers[0]).not.toHaveProperty("opacity");
    expect(workspace.record.scenery.layers[0]).not.toHaveProperty("zIndex");
  });

  it("uses empty layers when Core scenery content is absent", () => {
    const detail = sceneryDetail();
    delete detail.content;

    const workspace = toCoreSceneryAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail,
      records: [],
    });

    expect(workspace.record).toMatchObject({
      mode: "scenery",
      scenery: { layers: [] },
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

  it("handles empty UI Set content and zero-sized canvases", () => {
    const empty = uiSetDetail();
    empty.dimensions = { width: 0, height: 0 };
    empty.content = {
      components: [
        {
          id: 8,
          name: "Fallback",
          size: { width: 20, height: 10 },
          position: { x: 5, y: 3 },
          metadata: { kind: "unknown" },
        },
      ],
    };
    const noContent = uiSetDetail();
    delete noContent.content;

    const emptyWorkspace = toCoreUISetAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail: empty,
      records: [],
    });
    const noContentWorkspace = toCoreUISetAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail: noContent,
      records: [],
    });

    expect(emptyWorkspace.record).toMatchObject({
      mode: "uiset",
      uiset: {
        components: [
          { kind: "panel", bounds: { x: 0, y: 0, width: 0, height: 0 } },
        ],
      },
    });
    expect(noContentWorkspace.record).toMatchObject({
      mode: "uiset",
      uiset: { components: [] },
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

function spriteDetail(
  type: "character" | "object",
): Extract<AssetDetailResponse, { type: "character" | "object" }> {
  return {
    assetId: 9,
    projectId: 11,
    name: type === "character" ? "Hero" : "Chest",
    description: "A persisted sprite asset",
    type,
    perspective: "Top-Down",
    dimensions: { width: 32, height: 32 },
    tags: [],
    version: 1,
    content: { directionCount: 4, prototype: [] },
  } as unknown as Extract<
    AssetDetailResponse,
    { type: "character" | "object" }
  >;
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
