import { describe, expect, it } from "vitest";

import type { ProjectAsset } from "../types";
import {
  createDefaultAssetRecord,
  mergeAssetRecord,
} from "./mock/record-defaults";
import { isAssetRecordForKind } from "./record.validation";

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

describe("asset record kind boundaries", () => {
  it("accepts relative and HTTPS image URLs", () => {
    expect(
      isAssetRecordForKind("scenery", {
        mode: "scenery",
        prompt: "Forest",
        scenery: {
          layers: [
            {
              id: "sky",
              label: "Sky",
              detail: "Background",
              imageUrl: "/assets/sky.png",
              blendMode: "normal",
            },
            {
              id: "mist",
              label: "Mist",
              detail: "Atmosphere",
              imageUrl: "https://cdn.example.com/mist.png",
              blendMode: "multiply",
            },
          ],
        },
      }),
    ).toBe(true);
  });

  it("rejects unsafe image URL schemes", () => {
    const scenery = {
      mode: "scenery",
      prompt: "Forest",
      scenery: {
        layers: [
          {
            id: "sky",
            label: "Sky",
            detail: "Background",
            imageUrl: "javascript:alert(1)",
            blendMode: "normal",
          },
        ],
      },
    };
    const tileset = {
      mode: "tileset",
      prompt: "Props",
      tileset: {
        gridSize: 8,
        items: [
          {
            id: "barrel",
            label: "Barrel",
            imageUrl: "data:image/png;base64,abc",
            tiles: [[0, 0]],
          },
        ],
      },
    };

    expect(isAssetRecordForKind("scenery", scenery)).toBe(false);
    expect(isAssetRecordForKind("tileset", tileset)).toBe(false);
  });

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

  it("does not accept one sprite record kind as the other", () => {
    const characterRecord = createDefaultAssetRecord("character", asset);
    const objectRecord = createDefaultAssetRecord("object", asset);

    expect(isAssetRecordForKind("character", characterRecord)).toBe(true);
    expect(isAssetRecordForKind("object", objectRecord)).toBe(true);
    expect(isAssetRecordForKind("object", characterRecord)).toBe(false);
    expect(isAssetRecordForKind("character", objectRecord)).toBe(false);
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
