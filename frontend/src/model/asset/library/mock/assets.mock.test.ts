import { beforeEach, describe, expect, it } from "vitest";

import {
  copyMockAsset,
  deleteMockAsset,
  listMockAssetGroups,
  resetMockAssets,
  updateMockAsset,
} from "./assets.mock";

describe("mock asset library operations", () => {
  beforeEach(() => resetMockAssets());

  it("seeds only the intended Moonlit Orchard assets", async () => {
    const groups = await listMockAssetGroups("moonlit-orchard");
    const names = groups.flatMap((group) =>
      group.assets.map((asset) => asset.name),
    );

    expect(names).toEqual([
      "Swordsman",
      "Knight",
      "Alchemy Table",
      "Orchard Ground Set",
      "Moonlit Orchard Scene",
      "Quest Log",
    ]);
  });

  it("copies an asset next to its source with independent records", async () => {
    const groups = await copyMockAsset("moonlit-orchard", "swordsman");
    const characters = groups.find((group) => group.kind === "character");
    const source = characters?.assets[0];
    const copy = characters?.assets[1];

    expect(copy).toMatchObject({
      name: "Swordsman Copy",
      thumbnailUrl: "/assets/characters/swordsman/prototype.png",
    });
    expect(copy?.id).not.toBe(source?.id);
    expect(copy?.history[0]?.id).not.toBe(source?.history[0]?.id);
  });

  it("seeds the Alchemy Table with matching metadata", async () => {
    const groups = await listMockAssetGroups("moonlit-orchard");
    const alchemyTable = groups
      .find((group) => group.kind === "object")
      ?.assets.find((asset) => asset.id === "alchemy-table");

    expect(alchemyTable).toMatchObject({
      name: "Alchemy Table",
      description: "48x64 alchemy workstation sprite",
      canvasSize: "48 × 64 px",
      tags: ["alchemy", "table", "workstation", "pixel-art"],
      previewCrop: { width: 48, height: 64 },
    });
  });

  it("deletes only the requested project asset", async () => {
    await deleteMockAsset("moonlit-orchard", "swordsman");
    const groups = await listMockAssetGroups("moonlit-orchard");
    const ids = groups.flatMap((group) =>
      group.assets.map((asset) => asset.id),
    );

    expect(ids).not.toContain("swordsman");
    expect(ids).toContain("knight");
  });

  it("persists metadata updates without changing other assets", async () => {
    const groups = await updateMockAsset("moonlit-orchard", "swordsman", {
      name: "Veteran Swordsman",
      description: "A battle-worn four-direction character",
      tags: ["hero", "veteran"],
      canvasSize: "128 × 128 px",
      perspective: "Isometric",
    });
    const characters = groups.find((group) => group.kind === "character");

    expect(characters?.assets[0]).toMatchObject({
      id: "swordsman",
      name: "Veteran Swordsman",
      description: "A battle-worn four-direction character",
      tags: ["hero", "veteran"],
      canvasSize: "128 × 128 px",
      perspective: "Isometric",
    });
    expect(characters?.assets[1]?.name).toBe("Knight");
    expect(await listMockAssetGroups("moonlit-orchard")).toEqual(groups);
  });

  it("rejects metadata updates for missing assets", async () => {
    await expect(
      updateMockAsset("moonlit-orchard", "missing", {
        name: "Missing",
        description: "",
        tags: [],
        canvasSize: "64 × 64 px",
        perspective: "Top-down",
      }),
    ).rejects.toMatchObject({
      code: "NOT_FOUND",
      message: "Asset was not found.",
    });
  });
});
