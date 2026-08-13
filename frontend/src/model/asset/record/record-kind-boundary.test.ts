import { describe, expect, it } from "vitest";

import type { ProjectAsset } from "../types";
import {
  createDefaultAssetRecord,
  mergeAssetRecord,
} from "./mock/record-defaults";

const asset: ProjectAsset = {
  id: "test-sprite",
  name: "Test sprite",
  description: "A test sprite",
  version: "v1",
  canvasSize: "32 x 32 px",
  perspective: "Top-Down",
  tags: [],
  history: [],
  animations: [],
};

describe("asset record defaults", () => {
  it("creates independent character and object records from shared sprite data", () => {
    const characterRecord = createDefaultAssetRecord("character", asset);
    const objectRecord = createDefaultAssetRecord("object", asset);

    expect(characterRecord).toMatchObject({
      mode: "character",
      character: { nodePositions: {} },
    });
    expect(objectRecord).toMatchObject({
      mode: "object",
      object: { nodePositions: {} },
    });
    expect(characterRecord).not.toHaveProperty("object");
    expect(objectRecord).not.toHaveProperty("character");
  });

  it("leaves animations empty when an asset has no animation data", () => {
    const characterRecord = createDefaultAssetRecord("character", asset);
    const objectRecord = createDefaultAssetRecord("object", asset);

    expect(characterRecord.character.animations).toEqual([]);
    expect(objectRecord.object.animations).toEqual([]);
  });

  it("creates one prototype frame per remote direction image", () => {
    const directionalAsset = {
      ...asset,
      prototypeUrls: ["/front.png", "/back.png", "/left.png", "/right.png"],
    };
    const record = createDefaultAssetRecord("character", directionalAsset);

    expect(record.character.prototype).toMatchObject({
      imageUrl: "/front.png",
      frameUrls: directionalAsset.prototypeUrls,
      columns: 2,
      rows: 2,
    });
  });

  it("merges object records without crossing the character boundary", () => {
    const fallback = createDefaultAssetRecord("object", asset);
    const saved = structuredClone(fallback);
    saved.prompt = "Saved object";
    saved.object.nodePositions = { prototype: { x: 12, y: 24 } };

    expect(mergeAssetRecord("object", fallback, saved)).toMatchObject({
      mode: "object",
      prompt: "Saved object",
      object: { nodePositions: { prototype: { x: 12, y: 24 } } },
    });
  });
});
