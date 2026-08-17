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
  loadCoreSpriteAssetWorkspace,
  saveCoreSpriteAssetRevision,
  toCoreSpriteAssetWorkspace,
} from "./core-sprite-record";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.projectDetail.mockResolvedValue({ name: "Demo" });
  mocks.assetRecords.mockResolvedValue({ records: [] });
});

describe("saveCoreSpriteAssetRevision", () => {
  it("persists a Core sprite revision with its loaded base version", async () => {
    mocks.assetRecord.mockResolvedValue({ version: 4 });
    mocks.assetRecords.mockResolvedValue({ records: [] });

    await expect(
      saveCoreSpriteAssetRevision({
        projectId: "11",
        assetId: "9",
        version: "v3",
        record: {
          mode: "character",
          prompt: "Hero",
          character: {
            prototype: {
              format: "png-sprite-sheet",
              imageUrl: "/front.png",
              frameUrls: ["/front.png", "/back.png"],
              frameWidth: 32,
              frameHeight: 32,
              columns: 2,
              rows: 1,
            },
            animations: [],
            nodePositions: { prototype: { x: 10, y: 20 } },
          },
        },
      }),
    ).resolves.toMatchObject({ version: "v4" });

    expect(mocks.assetRecord).toHaveBeenCalledWith({
      assetId: 9,
      expectedVersion: 3,
      content: expect.objectContaining({
        directionCount: 2,
        metadata: { nodePositions: { prototype: { x: 10, y: 20 } } },
      }),
    });
  });
});

describe("loadCoreSpriteAssetWorkspace", () => {
  it.each(["draft", "0", "1.5"])(
    "skips a non-persisted asset ID: %s",
    async (assetId) => {
      await expect(
        loadCoreSpriteAssetWorkspace({ projectId: "11", assetId }),
      ).resolves.toBeUndefined();
      expect(mocks.assetDetail).not.toHaveBeenCalled();
    },
  );

  it("skips Core assets without a sprite workspace", async () => {
    mocks.assetDetail.mockResolvedValue(sceneryDetail());

    await expect(
      loadCoreSpriteAssetWorkspace({ projectId: "11", assetId: "9" }),
    ).resolves.toBeUndefined();
    expect(mocks.projectDetail).not.toHaveBeenCalled();
    expect(mocks.assetRecords).not.toHaveBeenCalled();
  });

  it("loads a persisted object with its project and record history", async () => {
    mocks.assetDetail.mockResolvedValue(objectDetail());

    await expect(
      loadCoreSpriteAssetWorkspace({ projectId: "11", assetId: "9" }),
    ).resolves.toMatchObject({
      projectName: "Demo",
      asset: { id: "9", kind: "object" },
      record: { mode: "object" },
    });
    expect(mocks.assetDetail).toHaveBeenCalledWith(9);
    expect(mocks.projectDetail).toHaveBeenCalledWith("11");
    expect(mocks.assetRecords).toHaveBeenCalledWith(9);
  });
});

describe("toCoreSpriteAssetWorkspace", () => {
  it("maps Core animation frames into an editor sprite record", () => {
    const workspace = toCoreSpriteAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail: characterDetail(),
      records: [
        {
          recordId: 31,
          assetId: 9,
          version: 3,
          contentId: 41,
          createdAt: "2026-08-14T01:00:00Z",
          name: "Hero",
          description: "A generated hero",
          perspective: "Top-Down",
          dimensions: { width: 32, height: 32 },
        },
      ] satisfies AssetRecordResponse<"character">[],
    });

    expect(workspace).toMatchObject({
      projectName: "Demo",
      asset: {
        id: "9",
        projectId: "11",
        kind: "character",
        version: "v3",
        history: [
          {
            id: "31",
            version: "v3",
            status: "ready",
            isCurrent: true,
          },
        ],
      },
      record: {
        mode: "character",
        prompt: "A generated hero",
        character: {
          prototype: {
            imageUrl: "/front.png",
            frameUrls: ["/front.png", "/right.png", "/back.png", "/left.png"],
            frameWidth: 32,
            frameHeight: 32,
            columns: 2,
            rows: 2,
          },
          animations: [
            {
              kind: "clip",
              id: "7",
              label: "Walk",
              frameCount: 2,
              spriteSheet: {
                imageUrl: "/walk-1.png",
                frameUrls: ["/walk-1.png", "/walk-2.png"],
                frameWidth: 32,
                frameHeight: 32,
                columns: 2,
                rows: 1,
              },
            },
          ],
          nodePositions: {},
        },
      },
    });
  });

  it("rejects non-sprite Core assets", () => {
    expect(() =>
      toCoreSpriteAssetWorkspace({
        projectId: "11",
        projectName: "Demo",
        detail: sceneryDetail(),
        records: [],
      }),
    ).toThrow("require a Character or Object asset");
  });

  it("omits independent frame metadata when no usable frames exist", () => {
    const detail = objectDetail();
    if (detail.type !== "object") throw new Error("Expected object detail");
    detail.content = {
      directionCount: 2,
      prototype: [{ id: 1 }],
      animations: [{ id: 8, name: "Idle", frames: [{ id: 2 }] }],
    };

    const workspace = toCoreSpriteAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail,
      records: [],
    });

    expect(workspace.record).toMatchObject({
      mode: "object",
      object: {
        prototype: { imageUrl: "" },
        animations: [{ id: "8", frameCount: 0 }],
      },
    });
    if (workspace.record.mode !== "object") return;
    expect(workspace.record.object.prototype).not.toHaveProperty("frameUrls");
    expect(workspace.record.object.animations?.[0]).not.toHaveProperty(
      "spriteSheet",
    );
  });
});

function characterDetail(): AssetDetailResponse {
  return {
    assetId: 9,
    projectId: 11,
    name: "Hero",
    description: "A generated hero",
    type: "character",
    perspective: "Top-Down",
    dimensions: { width: 32, height: 32 },
    tags: [],
    version: 3,
    content: {
      directionCount: 4,
      prototype: [
        { id: 1, url: "/front.png" },
        { id: 2, url: "/right.png" },
        { id: 3, url: "/back.png" },
        { id: 4, url: "/left.png" },
      ],
      animations: [
        {
          id: 7,
          name: "Walk",
          frames: [
            { id: 1, url: "/walk-1.png", duration: 83 },
            { id: 2, url: "/walk-2.png", duration: 83 },
          ],
        },
      ],
    },
  };
}

function objectDetail(): AssetDetailResponse {
  return { ...characterDetail(), type: "object" } as AssetDetailResponse;
}

function sceneryDetail(): AssetDetailResponse {
  return {
    ...characterDetail(),
    type: "scenery",
    content: undefined,
  } as AssetDetailResponse;
}
