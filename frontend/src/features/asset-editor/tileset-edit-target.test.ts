import { describe, expect, it } from "vitest";

import type { TilesetItem } from "@/model";

import {
  MAX_TILESET_EDIT_TARGETS,
  resolveTilesetEditTarget,
} from "./tileset-edit-target";

const items: TilesetItem[] = [
  {
    id: "sofa",
    label: "Sofa",
    tiles: [
      [0, 0],
      [1, 0],
      [0, 1],
      [1, 1],
    ],
  },
  { id: "lamp", label: "Lamp", tiles: [[3, 3]] },
];

describe("resolveTilesetEditTarget", () => {
  it("resolves one complete item to an item edit", () => {
    expect(
      resolveTilesetEditTarget({
        selectedCellIndexes: [5, 1, 4, 0, 0, 2],
        items,
        gridSize: 4,
      }),
    ).toEqual({
      target: {
        kind: "item",
        itemId: "sofa",
        label: "Sofa",
        position: [0, 0],
        positions: [
          [0, 0],
          [1, 0],
          [0, 1],
          [1, 1],
        ],
      },
      error: null,
    });
  });

  it("resolves partial and cross-item selections to stable tile targets", () => {
    expect(
      resolveTilesetEditTarget({
        selectedCellIndexes: [15, 1, 0, 15, 2, -1, 1.5],
        items,
        gridSize: 4,
      }).target,
    ).toMatchObject({
      kind: "tiles",
      label: "Sofa / Tile 1, Sofa / Tile 2, Lamp / Tile 1",
      positions: [
        [0, 0],
        [1, 0],
        [3, 3],
      ],
    });
  });

  it("rejects empty and oversized editable selections", () => {
    expect(
      resolveTilesetEditTarget({
        selectedCellIndexes: [2, 9],
        items,
        gridSize: 4,
      }),
    ).toEqual({ target: null, error: "missing" });

    const manyTiles: TilesetItem[] = [
      {
        id: "many",
        label: "Many",
        tiles: Array.from(
          { length: MAX_TILESET_EDIT_TARGETS + 1 },
          (_, index) =>
            [index % 17, Math.floor(index / 17)] as [number, number],
        ),
      },
    ];
    expect(
      resolveTilesetEditTarget({
        selectedCellIndexes: Array.from(
          { length: MAX_TILESET_EDIT_TARGETS + 1 },
          (_, index) => index,
        ),
        items: manyTiles,
        gridSize: 17,
      }),
    ).toEqual({ target: null, error: "too-many" });
  });

  it("does not infer an item when items share the same footprint", () => {
    expect(
      resolveTilesetEditTarget({
        selectedCellIndexes: [0],
        items: [
          { id: "first", label: "First", tiles: [[0, 0]] },
          { id: "second", label: "Second", tiles: [[0, 0]] },
        ],
        gridSize: 4,
      }).target,
    ).toMatchObject({ kind: "tiles", positions: [[0, 0]] });
  });
});
