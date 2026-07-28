import { useReducer } from "react";

import type { EditorTilesetItem } from "../../domain";

import type { TilesetCanvasEvent } from "./TilesetCanvas.interface";

type ItemCellSelection = {
  type: "item-cell";
  itemId: string;
  cellIndex: number;
};

type CanvasCellSelection = {
  type: "canvas-cell";
  cellIndex: number;
};

type TilesetSelection = ItemCellSelection | CanvasCellSelection;

export type TilesetCanvasState = {
  selectedItems: string[];
  selectedTargets: TilesetSelection[];
};

export type TilesetCanvasStateEvent =
  | { type: "item.toggle"; itemId: string }
  | { type: "item-cell.toggle"; itemId: string; cellIndex: number }
  | TilesetCanvasEvent;

export const initialTilesetCanvasState: TilesetCanvasState = {
  selectedItems: [],
  selectedTargets: [],
};

export function reduceTilesetCanvas(
  state: TilesetCanvasState,
  event: TilesetCanvasStateEvent,
  items: EditorTilesetItem[],
  gridSize: number,
): TilesetCanvasState {
  if (event.type === "item.toggle") {
    const selected = state.selectedItems.includes(event.itemId);
    return {
      selectedItems: selected
        ? state.selectedItems.filter((item) => item !== event.itemId)
        : [...state.selectedItems, event.itemId],
      selectedTargets: selected
        ? state.selectedTargets
        : state.selectedTargets.filter(
            (target) =>
              target.type !== "item-cell" || target.itemId !== event.itemId,
          ),
    };
  }

  if (event.type === "cell.selection.toggled") {
    const target = findCellTarget(event.cellIndex, items, gridSize);
    return target.type === "item-cell"
      ? reduceTilesetCanvas(
          state,
          { ...target, type: "item-cell.toggle" },
          items,
          gridSize,
        )
      : {
          ...state,
          selectedTargets: toggleSelection(state.selectedTargets, target),
        };
  }

  const target: ItemCellSelection = {
    type: "item-cell",
    itemId: event.itemId,
    cellIndex: event.cellIndex,
  };
  const item = items.find((candidate) => candidate.id === target.itemId);
  const itemTargets = item
    ? item.tiles.map((_, cellIndex) => ({
        type: "item-cell" as const,
        itemId: item.id,
        cellIndex,
      }))
    : [];

  if (item && state.selectedItems.includes(target.itemId)) {
    return {
      selectedItems: state.selectedItems.filter((id) => id !== target.itemId),
      selectedTargets: [
        ...state.selectedTargets.filter(
          (selection) =>
            selection.type !== "item-cell" ||
            selection.itemId !== target.itemId,
        ),
        ...itemTargets.filter(
          (itemTarget) => !sameSelection(itemTarget, target),
        ),
      ],
    };
  }

  const restoresItem =
    item &&
    !hasSelection(state.selectedTargets, target) &&
    itemTargets.length > 0 &&
    itemTargets.every(
      (itemTarget) =>
        sameSelection(itemTarget, target) ||
        hasSelection(state.selectedTargets, itemTarget),
    );
  if (restoresItem) {
    return {
      selectedItems: [...state.selectedItems, target.itemId],
      selectedTargets: state.selectedTargets.filter(
        (selection) =>
          selection.type !== "item-cell" || selection.itemId !== target.itemId,
      ),
    };
  }

  return {
    ...state,
    selectedTargets: toggleSelection(state.selectedTargets, target),
  };
}

export function useTilesetCanvasStateMachine(
  items: EditorTilesetItem[],
  gridSize: number,
) {
  const [state, dispatch] = useReducer(
    (current: TilesetCanvasState, event: TilesetCanvasStateEvent) =>
      reduceTilesetCanvas(current, event, items, gridSize),
    initialTilesetCanvasState,
  );

  return {
    selectedItems: state.selectedItems,
    selectedCells: getSelectedCells(state, items, gridSize),
    selectedLabels: getSelectedLabels(state, items),
    isCellSelected: (itemId: string, cellIndex: number) =>
      state.selectedItems.includes(itemId) ||
      hasSelection(state.selectedTargets, {
        type: "item-cell",
        itemId,
        cellIndex,
      }),
    send: (event: TilesetCanvasStateEvent) => dispatch(event),
  };
}

function getSelectedCells(
  state: TilesetCanvasState,
  items: EditorTilesetItem[],
  gridSize: number,
) {
  return [
    ...state.selectedItems.flatMap((itemId) =>
      getItemCells(
        items.find((item) => item.id === itemId),
        gridSize,
      ),
    ),
    ...state.selectedTargets.flatMap((target) => {
      if (target.type === "canvas-cell") return [target.cellIndex];
      return (
        getItemCells(
          items.find((item) => item.id === target.itemId),
          gridSize,
        )[target.cellIndex] ?? []
      );
    }),
  ];
}

function getSelectedLabels(
  state: TilesetCanvasState,
  items: EditorTilesetItem[],
) {
  return [
    ...state.selectedItems.map(
      (itemId) => items.find((item) => item.id === itemId)?.label ?? itemId,
    ),
    ...state.selectedTargets.map((target) => {
      if (target.type === "canvas-cell") return `Tile ${target.cellIndex + 1}`;
      const item = items.find((candidate) => candidate.id === target.itemId);
      return `${item?.label ?? target.itemId} / Tile ${target.cellIndex + 1}`;
    }),
  ];
}

function findCellTarget(
  cellIndex: number,
  items: EditorTilesetItem[],
  gridSize: number,
): TilesetSelection {
  const item = items.find((candidate) =>
    getItemCells(candidate, gridSize).includes(cellIndex),
  );
  if (!item) return { type: "canvas-cell", cellIndex };

  return {
    type: "item-cell",
    itemId: item.id,
    cellIndex: getItemCells(item, gridSize).indexOf(cellIndex),
  };
}

function getItemCells(item: EditorTilesetItem | undefined, gridSize: number) {
  return item?.tiles.map(([x, y]) => y * gridSize + x) ?? [];
}

function toggleSelection(
  selections: TilesetSelection[],
  target: TilesetSelection,
) {
  return hasSelection(selections, target)
    ? selections.filter((selection) => !sameSelection(selection, target))
    : [...selections, target];
}

function hasSelection(
  selections: TilesetSelection[],
  target: TilesetSelection,
) {
  return selections.some((selection) => sameSelection(selection, target));
}

function sameSelection(left: TilesetSelection, right: TilesetSelection) {
  if (left.type === "item-cell" && right.type === "item-cell") {
    return left.itemId === right.itemId && left.cellIndex === right.cellIndex;
  }
  return (
    left.type === "canvas-cell" &&
    right.type === "canvas-cell" &&
    left.cellIndex === right.cellIndex
  );
}
