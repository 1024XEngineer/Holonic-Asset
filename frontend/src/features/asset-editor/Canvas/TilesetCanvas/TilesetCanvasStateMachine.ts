import { useMemo, useReducer } from "react";

import type { ItemTile, TilesetItem } from "@/model";

import type { TilesetCanvasEvent } from "./TilesetCanvas.interface";

export type TilesetCanvasState = {
  modelKey: string;
  selectedCellIndexes: number[];
};

export type TilesetCanvasStateEvent =
  | TilesetCanvasEvent
  | { type: "selection.cleared" }
  | { type: "item.toggle"; itemId: string }
  | { type: "item-cell.toggle"; itemId: string; itemCellIndex: number };

export type TilesetCanvasSelection = {
  selectedItems: string[];
  selectedCellIndexes: number[];
  selectedLabels: string[];
};

export function createInitialTilesetCanvasState(
  items: readonly TilesetItem[],
  gridSize: number,
): TilesetCanvasState {
  return {
    modelKey: getTilesetModelKey(items, gridSize),
    selectedCellIndexes: [],
  };
}

export function reduceTilesetCanvas(
  state: TilesetCanvasState,
  event: TilesetCanvasStateEvent,
  items: readonly TilesetItem[],
  gridSize: number,
): TilesetCanvasState {
  return reduceTilesetCanvasWithIndex(
    state,
    event,
    createTilesetCanvasIndex(items, gridSize),
  );
}

function reduceTilesetCanvasWithIndex(
  state: TilesetCanvasState,
  event: TilesetCanvasStateEvent,
  index: TilesetCanvasIndex,
): TilesetCanvasState {
  const current =
    state.modelKey === index.modelKey
      ? state
      : { modelKey: index.modelKey, selectedCellIndexes: [] };

  switch (event.type) {
    case "item.toggle": {
      const item = index.itemsById.get(event.itemId);
      const itemCells = item ? (index.itemCellIndexes.get(item) ?? []) : [];
      if (itemCells.length === 0) return current;

      const selected = new Set(current.selectedCellIndexes);
      const itemSelected = itemCells.every((cellIndex) =>
        selected.has(cellIndex),
      );
      for (const cellIndex of itemCells) {
        if (itemSelected) selected.delete(cellIndex);
        else selected.add(cellIndex);
      }

      return withSelectedCells(current, selected);
    }
    case "item-cell.toggle": {
      const item = index.itemsById.get(event.itemId);
      const coordinate = item?.tiles[event.itemCellIndex];
      const cellIndex = coordinate
        ? getTilesetCellIndex(coordinate, index.gridSize)
        : undefined;
      if (cellIndex === undefined) return current;

      return toggleCell(current, cellIndex);
    }
    case "selection.cleared":
      return withSelectedCells(current, new Set());
    case "generation-review.resolved":
      return current;
    case "cell.selection.toggled":
      if (!isTilesetCellIndex(event.gridCellIndex, index.gridSize))
        return current;
      return toggleCell(current, event.gridCellIndex);
  }
}

export function getTilesetCanvasSelection(
  state: TilesetCanvasState,
  items: readonly TilesetItem[],
  gridSize: number,
): TilesetCanvasSelection {
  return getTilesetCanvasSelectionWithIndex(
    state,
    createTilesetCanvasIndex(items, gridSize),
  );
}

function getTilesetCanvasSelectionWithIndex(
  state: TilesetCanvasState,
  index: TilesetCanvasIndex,
): TilesetCanvasSelection {
  const selectedCellIndexes =
    state.modelKey === index.modelKey
      ? uniqueSorted(
          state.selectedCellIndexes.filter((cellIndex) =>
            isTilesetCellIndex(cellIndex, index.gridSize),
          ),
        )
      : [];
  const selectedCellSet = new Set(selectedCellIndexes);
  const selectedItems = index.items
    .filter((item) => {
      const itemCells = index.itemCellIndexes.get(item) ?? [];
      return (
        itemCells.length > 0 &&
        itemCells.every((cellIndex) => selectedCellSet.has(cellIndex))
      );
    })
    .map((item) => item.id);
  const selectedItemSet = new Set(selectedItems);
  const selectedLabels = [
    ...selectedItems.map(
      (itemId) => index.itemsById.get(itemId)?.label ?? itemId,
    ),
    ...selectedCellIndexes.flatMap((cellIndex) => {
      const owner = index.cellOwnersByIndex.get(cellIndex);
      if (!owner) return [`Tile ${cellIndex + 1}`];
      if (selectedItemSet.has(owner.item.id)) return [];
      return [`${owner.item.label} / Tile ${owner.itemCellIndex + 1}`];
    }),
  ];

  return { selectedItems, selectedCellIndexes, selectedLabels };
}

export function getTilesetCellIndex([column, row]: ItemTile, gridSize: number) {
  if (
    !isValidGridSize(gridSize) ||
    !Number.isInteger(column) ||
    !Number.isInteger(row) ||
    column < 0 ||
    row < 0 ||
    column >= gridSize ||
    row >= gridSize
  ) {
    return undefined;
  }

  return row * gridSize + column;
}

export function getTilesetItemCellIndexes(
  item: TilesetItem | undefined,
  gridSize: number,
) {
  if (!item) return [];

  return uniqueSorted(
    item.tiles.flatMap((coordinate) => {
      const cellIndex = getTilesetCellIndex(coordinate, gridSize);
      return cellIndex === undefined ? [] : [cellIndex];
    }),
  );
}

export function isValidGridSize(gridSize: number) {
  return Number.isInteger(gridSize) && gridSize > 0;
}

export function useTilesetCanvasStateMachine(
  items: readonly TilesetItem[],
  gridSize: number,
) {
  const index = useMemo(
    () => createTilesetCanvasIndex(items, gridSize),
    [items, gridSize],
  );
  const [state, dispatch] = useReducer(
    (
      current: TilesetCanvasState,
      event: TilesetCanvasStateEvent,
    ): TilesetCanvasState =>
      reduceTilesetCanvasWithIndex(current, event, index),
    undefined,
    () => ({ modelKey: index.modelKey, selectedCellIndexes: [] }),
  );
  const selection = useMemo(
    () => getTilesetCanvasSelectionWithIndex(state, index),
    [index, state],
  );
  const selectedCellSet = new Set(selection.selectedCellIndexes);

  return {
    ...selection,
    isCellSelected: (itemId: string, itemCellIndex: number) => {
      const item = index.itemsById.get(itemId);
      const coordinate = item?.tiles[itemCellIndex];
      const target = coordinate
        ? getTilesetCellIndex(coordinate, index.gridSize)
        : undefined;
      return target !== undefined && selectedCellSet.has(target);
    },
    send: dispatch,
  };
}

function toggleCell(state: TilesetCanvasState, cellIndex: number) {
  const selected = new Set(state.selectedCellIndexes);
  if (selected.has(cellIndex)) selected.delete(cellIndex);
  else selected.add(cellIndex);
  return withSelectedCells(state, selected);
}

function withSelectedCells(
  state: TilesetCanvasState,
  selectedCellIndexes: ReadonlySet<number>,
) {
  return {
    ...state,
    selectedCellIndexes: uniqueSorted([...selectedCellIndexes]),
  };
}

function isTilesetCellIndex(cellIndex: number, gridSize: number) {
  return (
    isValidGridSize(gridSize) &&
    Number.isInteger(cellIndex) &&
    cellIndex >= 0 &&
    cellIndex < gridSize * gridSize
  );
}

type TilesetCanvasIndex = {
  gridSize: number;
  modelKey: string;
  items: readonly TilesetItem[];
  itemsById: Map<string, TilesetItem>;
  itemCellIndexes: Map<TilesetItem, number[]>;
  cellOwnersByIndex: Map<number, { item: TilesetItem; itemCellIndex: number }>;
};

function createTilesetCanvasIndex(
  items: readonly TilesetItem[],
  gridSize: number,
): TilesetCanvasIndex {
  const itemsById = new Map<string, TilesetItem>();
  const itemCellIndexes = new Map<TilesetItem, number[]>();
  const cellOwnersByIndex = new Map<
    number,
    { item: TilesetItem; itemCellIndex: number }
  >();

  for (const item of items) {
    if (!itemsById.has(item.id)) itemsById.set(item.id, item);

    const cellIndexes = getTilesetItemCellIndexes(item, gridSize);
    itemCellIndexes.set(item, cellIndexes);

    item.tiles.forEach((coordinate, itemCellIndex) => {
      const cellIndex = getTilesetCellIndex(coordinate, gridSize);
      if (cellIndex !== undefined && !cellOwnersByIndex.has(cellIndex)) {
        cellOwnersByIndex.set(cellIndex, { item, itemCellIndex });
      }
    });
  }

  return {
    gridSize,
    modelKey: getTilesetModelKey(items, gridSize),
    items,
    itemsById,
    itemCellIndexes,
    cellOwnersByIndex,
  };
}

function getTilesetModelKey(items: readonly TilesetItem[], gridSize: number) {
  return JSON.stringify([gridSize, items.map((item) => [item.id, item.tiles])]);
}

function uniqueSorted(values: readonly number[]) {
  return [...new Set(values)].sort((left, right) => left - right);
}
