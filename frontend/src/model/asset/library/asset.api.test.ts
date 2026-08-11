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
