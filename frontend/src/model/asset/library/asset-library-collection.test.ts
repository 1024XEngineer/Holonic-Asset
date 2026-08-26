import { describe, expect, it } from "vitest";

import type { ProjectAsset } from "../types";
import { createAssetLibraryCollection } from "./asset-library-collection";
import type { AssetGroup } from "./types";

function asset(
  id: string,
  name: string,
  overrides: Partial<ProjectAsset> = {},
): ProjectAsset {
  return {
    id,
    name,
    description: `${name} description`,
    version: "v1",
    canvasSize: "64 x 64 px",
    perspective: "Top-Down",
    tags: [],
    history: [],
    animations: [],
    ...overrides,
  };
}

function assetGroups(): AssetGroup[] {
  return [
    {
      kind: "character",
      assets: [
        asset("asset-1", "Moonlit Swordsman", {
          tags: [
            { name: "hero", description: "", color: "#4F46E5" },
            { name: "pixel-art", description: "", color: "#4F46E5" },
          ],
        }),
      ],
    },
    {
      kind: "object",
      assets: [
        asset("asset-2", "Storage Barrel", {
          description: "Wooden prop",
          tags: [
            { name: "storage", description: "", color: "#4F46E5" },
            { name: "wood", description: "", color: "#4F46E5" },
          ],
        }),
      ],
    },
  ];
}

describe("createAssetLibraryCollection", () => {
  it("projects grouped assets into items and stable counts", () => {
    const groups = assetGroups();
    const collection = createAssetLibraryCollection(groups);

    expect(collection.items).toMatchObject([
      { id: "asset-1", kind: "character" },
      { id: "asset-2", kind: "object" },
    ]);
    expect(collection.counts).toEqual({
      character: 1,
      object: 1,
      tileset: 0,
      scenery: 0,
      uiset: 0,
      audio: 0,
    });
    expect(collection.totalAssets).toBe(2);
    expect(groups[0].assets[0]).not.toHaveProperty("kind");
  });

  it("finds an asset together with the kind of its group", () => {
    const collection = createAssetLibraryCollection(assetGroups());

    expect(collection.find("asset-2")).toMatchObject({
      id: "asset-2",
      kind: "object",
    });
    expect(collection.find("missing")).toBeUndefined();
  });

  it("appends to an existing group or creates a new group at the end", () => {
    const groups = assetGroups();
    const appended = createAssetLibraryCollection(groups).append(
      "character",
      asset("asset-3", "Knight"),
    );

    expect(appended[0].assets.map(({ id }) => id)).toEqual([
      "asset-1",
      "asset-3",
    ]);
    expect(appended[1]).toBe(groups[1]);

    const withAudio = createAssetLibraryCollection(groups).append(
      "audio",
      asset("asset-4", "Theme"),
    );
    expect(withAudio.at(-1)).toMatchObject({
      kind: "audio",
      assets: [{ id: "asset-4" }],
    });
  });

  it("inserts a derived asset next to its source without rebuilding other groups", () => {
    const groups = assetGroups();
    const updated = createAssetLibraryCollection(groups).insertAfter(
      "asset-1",
      (source) => ({ ...source, id: "asset-1-copy", name: "Copy" }),
    );

    expect(updated[0].assets.map(({ id }) => id)).toEqual([
      "asset-1",
      "asset-1-copy",
    ]);
    expect(updated[1]).toBe(groups[1]);
  });

  it("updates and removes only the located asset", () => {
    const groups = assetGroups();
    const updated = createAssetLibraryCollection(groups).update(
      "asset-2",
      (current) => ({ ...current, name: "Reinforced Barrel" }),
    );

    expect(updated[0]).toBe(groups[0]);
    expect(updated[1].assets[0].name).toBe("Reinforced Barrel");
    expect(groups[1].assets[0].name).toBe("Storage Barrel");

    const removed = createAssetLibraryCollection(updated).remove("asset-2");
    expect(removed[1]).toMatchObject({ kind: "object", assets: [] });
    expect(removed[0]).toBe(updated[0]);
  });

  it("uses one missing-asset error mode for every targeted write", () => {
    const collection = createAssetLibraryCollection(assetGroups(), {
      projectId: "project-1",
    });
    const operations = [
      () => collection.insertAfter("missing", (source) => source),
      () => collection.update("missing", (current) => current),
      () => collection.remove("missing"),
    ];

    for (const operation of operations) {
      expect(operation).toThrowError("Asset was not found.");
      expect(captureError(operation)).toMatchObject({
        code: "NOT_FOUND",
        details: { projectId: "project-1", assetId: "missing" },
      });
    }
  });

  it("protects asset identity on inserts and updates", () => {
    const collection = createAssetLibraryCollection(assetGroups());

    expect(() =>
      collection.append("object", asset("asset-1", "Duplicate")),
    ).toThrowError("Asset ID already exists.");
    expect(() =>
      collection.update("asset-1", (current) => ({
        ...current,
        id: "renamed",
      })),
    ).toThrowError("Asset updates cannot change asset identity.");
  });
});

function captureError(operation: () => unknown): unknown {
  try {
    operation();
  } catch (error) {
    return error;
  }
  throw new Error("Expected operation to throw.");
}
