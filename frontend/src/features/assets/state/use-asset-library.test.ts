import { describe, expect, it } from "vitest";

import type { AssetLibraryItem } from "@/model/asset";

import { filterAssetLibraryItems } from "./use-asset-library";

function asset(overrides: Partial<AssetLibraryItem> = {}): AssetLibraryItem {
  return {
    id: "asset-1",
    kind: "character",
    name: "Moonlit Swordsman",
    description: "Four-direction character",
    version: "v2",
    canvasSize: "64 x 64 px",
    perspective: "Top-Down",
    tags: ["hero", "pixel-art"],
    history: [],
    animations: [],
    ...overrides,
  };
}

const items: AssetLibraryItem[] = [
  asset(),
  asset({
    id: "asset-2",
    kind: "object",
    name: "Storage Barrel",
    description: "Wooden prop",
    tags: ["storage", "wood"],
  }),
];

describe("filterAssetLibraryItems", () => {
  it("searches asset tags and metadata case-insensitively", () => {
    expect(
      filterAssetLibraryItems(items, "PIXEL-ART", ["character", "object"]),
    ).toMatchObject([{ id: "asset-1", kind: "character" }]);
  });

  it("combines type selection with text search", () => {
    expect(filterAssetLibraryItems(items, "wood", ["character"])).toEqual([]);
    expect(filterAssetLibraryItems(items, "wood", ["object"])[0]?.id).toBe(
      "asset-2",
    );
  });
});
