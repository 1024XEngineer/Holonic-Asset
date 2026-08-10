import { describe, expect, it } from "vitest";

import type { TilesetItem } from "@/model";

import {
  createInitialTilesetCanvasState,
  getTilesetCanvasSelection,
  reduceTilesetCanvas,
} from "./TilesetCanvasStateMachine";

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

describe("tileset canvas state", () => {
  it("selects a whole item and degrades it to individual tiles", () => {
    let state = createInitialTilesetCanvasState(items, 4);
    state = reduceTilesetCanvas(
      state,
      { type: "item.toggle", itemId: "sofa" },
      items,
      4,
    );

    expect(getTilesetCanvasSelection(state, items, 4)).toEqual({
      selectedItems: ["sofa"],
      selectedCellIndexes: [0, 1, 4, 5],
      selectedLabels: ["Sofa"],
    });

    state = reduceTilesetCanvas(
      state,
      { type: "item-cell.toggle", itemId: "sofa", itemCellIndex: 1 },
      items,
      4,
    );
    expect(getTilesetCanvasSelection(state, items, 4)).toEqual({
      selectedItems: [],
      selectedCellIndexes: [0, 4, 5],
      selectedLabels: ["Sofa / Tile 1", "Sofa / Tile 3", "Sofa / Tile 4"],
    });
  });

  it("restores an item selection after its last missing tile is selected", () => {
    let state = createInitialTilesetCanvasState(items, 4);
    for (const cellIndex of [0, 1, 4, 5]) {
      state = reduceTilesetCanvas(
        state,
        { type: "cell.selection.toggled", gridCellIndex: cellIndex },
        items,
        4,
      );
    }

    expect(getTilesetCanvasSelection(state, items, 4)).toMatchObject({
      selectedItems: ["sofa"],
      selectedLabels: ["Sofa"],
    });
  });

  it("ignores invalid targets and clears stale selection across records", () => {
    let state = createInitialTilesetCanvasState(items, 4);
    state = reduceTilesetCanvas(
      state,
      { type: "cell.selection.toggled", gridCellIndex: 99 },
      items,
      4,
    );
    expect(state.selectedCellIndexes).toEqual([]);

    state = reduceTilesetCanvas(
      state,
      { type: "item.toggle", itemId: "lamp" },
      items,
      4,
    );
    expect(getTilesetCanvasSelection(state, [], 8)).toEqual({
      selectedItems: [],
      selectedCellIndexes: [],
      selectedLabels: [],
    });
  });

  it("does not confuse model keys when item ids contain delimiters", () => {
    const firstModel: TilesetItem[] = [
      { id: "a", label: "A", tiles: [[0, 0]] },
      { id: "b", label: "B", tiles: [[1, 0]] },
    ];
    const secondModel: TilesetItem[] = [
      { id: "a:0,0|b", label: "Combined", tiles: [[1, 0]] },
    ];
    let state = createInitialTilesetCanvasState(firstModel, 4);
    state = reduceTilesetCanvas(
      state,
      { type: "cell.selection.toggled", gridCellIndex: 0 },
      firstModel,
      4,
    );

    expect(getTilesetCanvasSelection(state, secondModel, 4)).toMatchObject({
      selectedCellIndexes: [],
      selectedLabels: [],
    });
  });
});
