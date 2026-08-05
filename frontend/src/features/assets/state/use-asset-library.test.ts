import { describe, expect, it } from "vitest";

import type { AssetGroup, ProjectAsset } from "@/model/asset";

import { countAssetsByKind, filterAssetGroups } from "./use-asset-library";

function asset(overrides: Partial<ProjectAsset> = {}): ProjectAsset {
  return {
    id: "asset-1",
    name: "Moonlit Swordsman",
    description: "Four-direction character",
    version: "v2",
    canvasSize: "64 × 64 px",
    perspective: "Top-down",
    tags: ["hero", "pixel-art"],
    history: [],
    animations: [],
    ...overrides,
  };
}

const groups: AssetGroup[] = [
  { kind: "character", assets: [asset()] },
  {
    kind: "object",
    assets: [
      asset({
        id: "asset-2",
        name: "Storage Barrel",
        description: "Wooden prop",
        tags: ["storage", "wood"],
      }),
    ],
  },
];

describe("filterAssetGroups", () => {
  it("searches asset tags and metadata without changing model data", () => {
    const result = filterAssetGroups(groups, "PIXEL-ART", [
      "character",
      "object",
    ]);

    expect(result).toHaveLength(1);
    expect(result[0]).toMatchObject({
      id: "asset-1",
      kind: "character",
      kindLabel: "Character",
    });
    expect(groups[0].assets[0]).not.toHaveProperty("kind");
  });

  it("combines type selection with text search", () => {
    expect(filterAssetGroups(groups, "wood", ["character"])).toEqual([]);
    expect(filterAssetGroups(groups, "wood", ["object"])[0]?.id).toBe(
      "asset-2",
    );
  });
});

describe("countAssetsByKind", () => {
  it("returns stable zero counts for empty asset kinds", () => {
    expect(countAssetsByKind(groups)).toEqual({
      character: 1,
      object: 1,
      tileset: 0,
      scenery: 0,
      ui: 0,
      audio: 0,
    });
  });
});
