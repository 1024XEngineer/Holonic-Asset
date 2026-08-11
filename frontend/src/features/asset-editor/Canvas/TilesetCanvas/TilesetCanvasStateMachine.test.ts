import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import type { TilesetItem } from "@/model";

import {
  createInitialTilesetCanvasState,
  getTilesetCellIndex,
  getTilesetCanvasSelection,
  getTilesetItemCellIndexes,
  isValidGridSize,
  reduceTilesetCanvas,
  useTilesetCanvasStateMachine,
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

  it("deselects whole items and ignores missing item targets", () => {
    let state = createInitialTilesetCanvasState(items, 4);
    state = reduceTilesetCanvas(
      state,
      { type: "item.toggle", itemId: "sofa" },
      items,
      4,
    );
    state = reduceTilesetCanvas(
      state,
      { type: "item.toggle", itemId: "sofa" },
      items,
      4,
    );

    expect(state.selectedCellIndexes).toEqual([]);
    expect(
      reduceTilesetCanvas(
        state,
        { type: "item.toggle", itemId: "missing" },
        items,
        4,
      ),
    ).toBe(state);
    expect(
      reduceTilesetCanvas(
        state,
        { type: "item-cell.toggle", itemId: "missing", itemCellIndex: 0 },
        items,
        4,
      ),
    ).toBe(state);
  });

  it("labels unowned cells and resets state against a new model", () => {
    let state = createInitialTilesetCanvasState(items, 4);
    state = reduceTilesetCanvas(
      state,
      { type: "cell.selection.toggled", gridCellIndex: 2 },
      items,
      4,
    );

    expect(getTilesetCanvasSelection(state, items, 4).selectedLabels).toEqual([
      "Tile 3",
    ]);

    const reset = reduceTilesetCanvas(
      state,
      { type: "item.toggle", itemId: "missing" },
      [],
      8,
    );
    expect(reset.selectedCellIndexes).toEqual([]);
    expect(reset.modelKey).not.toBe(state.modelKey);
  });

  it("validates grid sizes, coordinates, and item cells", () => {
    expect(isValidGridSize(4)).toBe(true);
    expect(isValidGridSize(0)).toBe(false);
    expect(isValidGridSize(1.5)).toBe(false);
    expect(getTilesetCellIndex([1, 2], 4)).toBe(9);
    expect(getTilesetCellIndex([0, 0], 0)).toBeUndefined();
    expect(getTilesetCellIndex([0.5, 0], 4)).toBeUndefined();
    expect(getTilesetCellIndex([0, 0.5], 4)).toBeUndefined();
    expect(getTilesetCellIndex([-1, 0], 4)).toBeUndefined();
    expect(getTilesetCellIndex([0, -1], 4)).toBeUndefined();
    expect(getTilesetCellIndex([4, 0], 4)).toBeUndefined();
    expect(getTilesetCellIndex([0, 4], 4)).toBeUndefined();
    expect(getTilesetItemCellIndexes(undefined, 4)).toEqual([]);
    expect(
      getTilesetItemCellIndexes(
        {
          id: "mixed",
          label: "Mixed",
          tiles: [
            [1, 1],
            [1, 1],
            [-1, 0],
            [4, 0],
          ],
        },
        4,
      ),
    ).toEqual([5]);
  });

  it("uses the first item and cell owner when ids and tiles overlap", () => {
    const overlappingItems: TilesetItem[] = [
      { id: "shared", label: "First", tiles: [[0, 0]] },
      {
        id: "shared",
        label: "Second",
        tiles: [
          [0, 0],
          [1, 0],
        ],
      },
    ];
    let state = createInitialTilesetCanvasState(overlappingItems, 4);
    state = reduceTilesetCanvas(
      state,
      { type: "item.toggle", itemId: "shared" },
      overlappingItems,
      4,
    );

    expect(getTilesetCanvasSelection(state, overlappingItems, 4)).toEqual({
      selectedItems: ["shared"],
      selectedCellIndexes: [0],
      selectedLabels: ["First"],
    });
  });

  it("reports unselected state for valid and missing item cells", () => {
    expect(
      renderToStaticMarkup(
        createElement(SelectionProbe, { itemId: "sofa", itemCellIndex: 0 }),
      ),
    ).toContain("false");
    expect(
      renderToStaticMarkup(
        createElement(SelectionProbe, {
          itemId: "missing",
          itemCellIndex: 0,
        }),
      ),
    ).toContain("false");
  });
});

function SelectionProbe({
  itemId,
  itemCellIndex,
}: {
  itemId: string;
  itemCellIndex: number;
}) {
  const canvas = useTilesetCanvasStateMachine(items, 4);
  return createElement(
    "span",
    null,
    String(canvas.isCellSelected(itemId, itemCellIndex)),
  );
}
