import { describe, expect, it } from "vitest";

import type { AssetListItemResponse } from "./asset.contract";
import { toAssetGroups } from "./asset.api";

const assetItemBase = {
  assetId: 1,
  projectId: 10,
  name: "Hero",
  description: "Main character",
  perspective: "Top-Down" as const,
  tags: [],
  version: 1,
};

describe("toAssetGroups", () => {
  it("maps API assets into typed groups with defaults and stable ordering", () => {
    expect(
      toAssetGroups([
        {
          assetId: 8,
          projectId: 1,
          name: "Barrel",
          description: "Wooden prop",
          dimensions: { width: 32, height: 32 },
          perspective: "Top-Down",
          type: "object",
          version: 2,
          tags: ["prop"],
        },
        {
          assetId: 9,
          projectId: 1,
          name: "Ground",
          description: "Tile set",
          dimensions: {
            tileSize: { width: 16, height: 16 },
            tileAmount: { columns: 8, rows: 8 },
          },
          perspective: "Top-Down",
          type: "tileSet",
          version: 1,
          tags: [],
        },
        {
          assetId: 10,
          projectId: 1,
          name: "Crate",
          description: "Another prop",
          dimensions: { width: 24, height: 24 },
          perspective: "Top-Down",
          type: "object",
          version: 3,
          tags: null,
        },
      ]),
    ).toMatchObject([
      {
        kind: "object",
        assets: [
          { id: "8", version: "v2", tags: ["prop"] },
          { id: "10", version: "v3", tags: [] },
        ],
      },
      {
        kind: "tileset",
        assets: [{ id: "9", version: "v1", tags: [] }],
      },
    ]);
  });

  it("maps backend perspective and dimensions into library assets", () => {
    const groups = toAssetGroups([
      {
        ...assetItemBase,
        type: "character",
        dimensions: { width: 48, height: 64 },
        perspective: "Side-On",
      },
      {
        ...assetItemBase,
        assetId: 2,
        type: "tileSet",
        dimensions: {
          tileSize: { width: 16, height: 24 },
          tileAmount: { columns: 8, rows: 6 },
        },
      },
    ] satisfies AssetListItemResponse[]);

    expect(groups).toMatchObject([
      {
        kind: "character",
        assets: [{ canvasSize: "48 × 64 px", perspective: "Side-On" }],
      },
      {
        kind: "tileset",
        assets: [{ canvasSize: "128 × 144 px", perspective: "Top-Down" }],
      },
    ]);
  });

  it("keeps every prototype direction URL for the editor", () => {
    const [group] = toAssetGroups([
      {
        ...assetItemBase,
        type: "character",
        dimensions: { width: 48, height: 64 },
        prototypeUrls: ["/front.png", "/back.png", "/left.png", "/right.png"],
      },
    ] satisfies AssetListItemResponse[]);

    expect(group.assets[0]).toMatchObject({
      thumbnailUrl: "/front.png",
      prototypeUrls: ["/front.png", "/back.png", "/left.png", "/right.png"],
    });
  });

  it("does not report a visual canvas size for audio assets", () => {
    const [group] = toAssetGroups([
      {
        ...assetItemBase,
        type: "audio",
        dimensions: null,
      },
    ] satisfies AssetListItemResponse[]);

    expect(group.assets[0].canvasSize).toBe("N/A");
  });
});
