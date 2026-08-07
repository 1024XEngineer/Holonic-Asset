import { useReducer } from "react";

import type { ItemTile } from "@/model";
import type { TilesetItem } from "@/model";

import type { TilesetCanvasEvent } from "./TilesetCanvas.interface";

export type TilesetCanvasState = {
  modelKey: string;
  selectedCellIndexes: number[];
};

export type TilesetCanvasStateEvent =
  | TilesetCanvasEvent
  | { type: "item.toggle"; itemId: string }
  | { type: "item-cell.toggle"; itemId: string; cellIndex: number };

export type TilesetCanvasSelection = {
  selectedItems: string[];
  selectedCells: number[];
  selectedLabels: string[];
};

export const initialTilesetCanvasState: TilesetCanvasState = {
  modelKey: "",
  selectedCellIndexes: [],
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
  const modelKey = getTilesetModelKey(items, gridSize);
  const current =
    state.modelKey === modelKey ? state : { modelKey, selectedCellIndexes: [] };

  if (event.type === "item.toggle") {
    const item = items.find((candidate) => candidate.id === event.itemId);
    const itemCells = getTilesetItemCellIndexes(item, gridSize);
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

  if (event.type === "item-cell.toggle") {
    const item = items.find((candidate) => candidate.id === event.itemId);
    const coordinate = item?.tiles[event.cellIndex];
    const cellIndex = coordinate
      ? getTilesetCellIndex(coordinate, gridSize)
      : undefined;
    if (cellIndex === undefined) return current;

    return toggleCell(current, cellIndex);
  }

  if (!isTilesetCellIndex(event.cellIndex, gridSize)) return current;
  return toggleCell(current, event.cellIndex);
}

export function getTilesetCanvasSelection(
  state: TilesetCanvasState,
  items: readonly TilesetItem[],
  gridSize: number,
): TilesetCanvasSelection {
  const modelKey = getTilesetModelKey(items, gridSize);
  const selectedCells =
    state.modelKey === modelKey
      ? uniqueSorted(
          state.selectedCellIndexes.filter((cellIndex) =>
            isTilesetCellIndex(cellIndex, gridSize),
          ),
        )
      : [];
  const selectedCellSet = new Set(selectedCells);
  const selectedItems = items
    .filter((item) => {
      const itemCells = getTilesetItemCellIndexes(item, gridSize);
      return (
        itemCells.length > 0 &&
        itemCells.every((cellIndex) => selectedCellSet.has(cellIndex))
      );
    })
    .map((item) => item.id);
  const selectedItemSet = new Set(selectedItems);
  const selectedLabels = [
    ...selectedItems.map(
      (itemId) => items.find((item) => item.id === itemId)?.label ?? itemId,
    ),
    ...selectedCells.flatMap((cellIndex) => {
      const owner = findCellOwner(cellIndex, items, gridSize);
      if (!owner) return [`Tile ${cellIndex + 1}`];
      if (selectedItemSet.has(owner.item.id)) return [];
      return [`${owner.item.label} / Tile ${owner.itemCellIndex + 1}`];
    }),
  ];

  return { selectedItems, selectedCells, selectedLabels };
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
  const [state, dispatch] = useReducer(
    (
      current: TilesetCanvasState,
      event: TilesetCanvasStateEvent,
    ): TilesetCanvasState =>
      reduceTilesetCanvas(current, event, items, gridSize),
    undefined,
    () => createInitialTilesetCanvasState(items, gridSize),
  );
  const selection = getTilesetCanvasSelection(state, items, gridSize);
  const selectedCells = new Set(selection.selectedCells);

  return {
    ...selection,
    isCellSelected: (itemId: string, cellIndex: number) => {
      const item = items.find((candidate) => candidate.id === itemId);
      const coordinate = item?.tiles[cellIndex];
      const target = coordinate
        ? getTilesetCellIndex(coordinate, gridSize)
        : undefined;
      return target !== undefined && selectedCells.has(target);
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

function findCellOwner(
  cellIndex: number,
  items: readonly TilesetItem[],
  gridSize: number,
) {
  for (const item of items) {
    const itemCellIndex = item.tiles.findIndex(
      (coordinate) => getTilesetCellIndex(coordinate, gridSize) === cellIndex,
    );
    if (itemCellIndex >= 0) return { item, itemCellIndex };
  }

  return undefined;
}

function isTilesetCellIndex(cellIndex: number, gridSize: number) {
  return (
    isValidGridSize(gridSize) &&
    Number.isInteger(cellIndex) &&
    cellIndex >= 0 &&
    cellIndex < gridSize * gridSize
  );
}

function getTilesetModelKey(items: readonly TilesetItem[], gridSize: number) {
  return `${gridSize}|${items
    .map(
      (item) =>
        `${item.id}:${item.tiles.map(([column, row]) => `${column},${row}`).join(";")}`,
    )
    .join("|")}`;
}

function uniqueSorted(values: readonly number[]) {
  return [...new Set(values)].sort((left, right) => left - right);
}
