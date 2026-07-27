import { useReducer } from "react";

import type { EditorSpriteSheetItem } from "@/features/assets/domain";

import type { SpriteSheetCanvasEvent } from "./SpriteSheetCanvas.interface";

type ItemCellSelection = {
  type: "item-cell";
  itemId: string;
  cellIndex: number;
};

type CanvasCellSelection = {
  type: "canvas-cell";
  cellIndex: number;
};

type SpriteSheetSelection = ItemCellSelection | CanvasCellSelection;

export type SpriteSheetCanvasState = {
  selectedItems: string[];
  selectedTargets: SpriteSheetSelection[];
};

export type SpriteSheetCanvasStateEvent =
  | { type: "item.toggle"; itemId: string }
  | { type: "item-cell.toggle"; itemId: string; cellIndex: number }
  | SpriteSheetCanvasEvent;

export const initialSpriteSheetCanvasState: SpriteSheetCanvasState = {
  selectedItems: [],
  selectedTargets: [],
};

export function reduceSpriteSheetCanvas(
  state: SpriteSheetCanvasState,
  event: SpriteSheetCanvasStateEvent,
  items: EditorSpriteSheetItem[],
  gridSize: number,
): SpriteSheetCanvasState {
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
      ? reduceSpriteSheetCanvas(
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
            selection.type !== "item-cell" || selection.itemId !== target.itemId,
        ),
        ...itemTargets.filter((itemTarget) => !sameSelection(itemTarget, target)),
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

export function useSpriteSheetCanvasStateMachine(
  items: EditorSpriteSheetItem[],
  gridSize: number,
) {
  const [state, dispatch] = useReducer(
    (current: SpriteSheetCanvasState, event: SpriteSheetCanvasStateEvent) =>
      reduceSpriteSheetCanvas(current, event, items, gridSize),
    initialSpriteSheetCanvasState,
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
    send: (event: SpriteSheetCanvasStateEvent) => dispatch(event),
  };
}

function getSelectedCells(
  state: SpriteSheetCanvasState,
  items: EditorSpriteSheetItem[],
  gridSize: number,
) {
  return [
    ...state.selectedItems.flatMap((itemId) =>
      getItemCells(items.find((item) => item.id === itemId), gridSize),
    ),
    ...state.selectedTargets.flatMap((target) => {
      if (target.type === "canvas-cell") return [target.cellIndex];
      return getItemCells(
        items.find((item) => item.id === target.itemId),
        gridSize,
      )[target.cellIndex] ?? [];
    }),
  ];
}

function getSelectedLabels(
  state: SpriteSheetCanvasState,
  items: EditorSpriteSheetItem[],
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
  items: EditorSpriteSheetItem[],
  gridSize: number,
): SpriteSheetSelection {
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

function getItemCells(item: EditorSpriteSheetItem | undefined, gridSize: number) {
  return item?.tiles.map(([x, y]) => y * gridSize + x) ?? [];
}

function toggleSelection(
  selections: SpriteSheetSelection[],
  target: SpriteSheetSelection,
) {
  return hasSelection(selections, target)
    ? selections.filter((selection) => !sameSelection(selection, target))
    : [...selections, target];
}

function hasSelection(
  selections: SpriteSheetSelection[],
  target: SpriteSheetSelection,
) {
  return selections.some((selection) => sameSelection(selection, target));
}

function sameSelection(left: SpriteSheetSelection, right: SpriteSheetSelection) {
  if (left.type === "item-cell" && right.type === "item-cell") {
    return left.itemId === right.itemId && left.cellIndex === right.cellIndex;
  }
  return left.type === "canvas-cell" && right.type === "canvas-cell" && left.cellIndex === right.cellIndex;
}
