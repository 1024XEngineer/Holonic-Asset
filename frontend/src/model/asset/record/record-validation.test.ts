import { describe, expect, it } from "vitest";

import type { AssetKind } from "../types";
import { isAssetRecordForKind } from "./record.validation";

const spriteSheet = {
  format: "png-sprite-sheet",
  imageUrl: "/sprite.png",
  frameWidth: 32,
  frameHeight: 32,
  columns: 4,
  rows: 2,
  row: 1,
} as const;

const spriteData = {
  prototype: spriteSheet,
  nodePositions: { idle: { x: 10, y: 20 } },
  animations: [
    {
      kind: "clip",
      id: "idle",
      label: "Idle",
      frameCount: 4,
      spriteSheet,
      audio: { label: "Footsteps", time: "00:01" },
    },
  ],
};

const validRecords: Record<AssetKind, unknown> = {
  character: { mode: "character", prompt: "Hero", character: spriteData },
  object: { mode: "object", prompt: "Chest", object: spriteData },
  scenery: {
    mode: "scenery",
    prompt: "Forest",
    scenery: {
      layers: [
        {
          id: "back",
          label: "Background",
          detail: "Trees",
          imageUrl: "/forest.png",
          blendMode: "normal",
        },
        {
          id: "light",
          label: "Light",
          detail: "Sunbeams",
          imageUrl: "/light.png",
          blendMode: "multiply",
        },
      ],
    },
  },
  tileset: {
    mode: "tileset",
    prompt: "Village",
    tileset: {
      gridSize: 16,
      items: [
        { id: "wall", label: "Wall", imageUrl: "/wall.png", tiles: [[0, 0]] },
        { id: "floor", label: "Floor", tiles: [[1, 2]] },
      ],
    },
  },
  uiset: {
    mode: "uiset",
    prompt: "Menu",
    uiset: {
      components: ["panel", "label", "button"].map((kind, index) => ({
        id: `${kind}-${index}`,
        label: kind,
        kind,
        bounds: { x: index, y: 2, width: 10, height: 5 },
      })),
    },
  },
  audio: { mode: "audio", prompt: "Theme", audio: {} },
};

describe("asset record validation", () => {
  it.each(Object.entries(validRecords) as [AssetKind, unknown][])(
    "accepts a valid %s record",
    (kind, record) => {
      expect(isAssetRecordForKind(kind, record)).toBe(true);
    },
  );

  it("rejects invalid record roots and mismatched kinds", () => {
    expect(isAssetRecordForKind("audio", null)).toBe(false);
    expect(isAssetRecordForKind("audio", [])).toBe(false);
    expect(
      isAssetRecordForKind("audio", { mode: "audio", prompt: 12, audio: {} }),
    ).toBe(false);
    expect(
      isAssetRecordForKind("audio", { mode: "unknown", prompt: "", audio: {} }),
    ).toBe(false);
    expect(isAssetRecordForKind("object", validRecords.character)).toBe(false);
  });

  it.each([
    ["missing sprite data", { mode: "character", prompt: "", character: null }],
    [
      "invalid sprite sheet dimensions",
      {
        mode: "character",
        prompt: "",
        character: { ...spriteData, prototype: { ...spriteSheet, columns: 0 } },
      },
    ],
    [
      "invalid sprite sheet row",
      {
        mode: "character",
        prompt: "",
        character: { ...spriteData, prototype: { ...spriteSheet, row: 2 } },
      },
    ],
    [
      "invalid node position",
      {
        mode: "character",
        prompt: "",
        character: { ...spriteData, nodePositions: { idle: { x: NaN, y: 2 } } },
      },
    ],
    [
      "duplicate animations",
      {
        mode: "character",
        prompt: "",
        character: {
          ...spriteData,
          animations: [spriteData.animations[0], spriteData.animations[0]],
        },
      },
    ],
    [
      "invalid animation audio",
      {
        mode: "character",
        prompt: "",
        character: {
          ...spriteData,
          animations: [
            { ...spriteData.animations[0], audio: { label: 1, time: "" } },
          ],
        },
      },
    ],
    [
      "invalid scenery layer",
      {
        mode: "scenery",
        prompt: "",
        scenery: {
          layers: [
            {
              id: "x",
              label: "x",
              detail: "",
              imageUrl: "",
              blendMode: "screen",
            },
          ],
        },
      },
    ],
    [
      "invalid tileset grid",
      { mode: "tileset", prompt: "", tileset: { gridSize: NaN, items: [] } },
    ],
    [
      "invalid tileset image",
      {
        mode: "tileset",
        prompt: "",
        tileset: {
          gridSize: 16,
          items: [{ id: "x", label: "x", imageUrl: 3, tiles: [] }],
        },
      },
    ],
    [
      "invalid tile coordinate",
      {
        mode: "tileset",
        prompt: "",
        tileset: {
          gridSize: 16,
          items: [{ id: "x", label: "x", tiles: [[0]] }],
        },
      },
    ],
    [
      "invalid UI component kind",
      {
        mode: "uiset",
        prompt: "",
        uiset: {
          components: [{ id: "x", label: "x", kind: "image", bounds: {} }],
        },
      },
    ],
    [
      "invalid UI component bounds",
      {
        mode: "uiset",
        prompt: "",
        uiset: {
          components: [
            {
              id: "x",
              label: "x",
              kind: "button",
              bounds: { x: 0, y: 0, width: Infinity, height: 10 },
            },
          ],
        },
      },
    ],
  ])("rejects %s", (_label, record) => {
    const kind = (record as { mode: AssetKind }).mode;
    expect(isAssetRecordForKind(kind, record)).toBe(false);
  });
});
