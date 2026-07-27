import { useReducer } from "react";

import type { EditorSpriteSheetItem } from "../../domain";

import type { SpriteSheetCanvasEvent } from "./SpriteSheetCanvas.interface";

type ItemTileSelection = {
  type: "item-tile";
  itemId: string;
  tileId: string;
};

type CanvasCellSelection = {
  type: "canvas-cell";
  cellIndex: number;
};

type SpriteSheetSelection = ItemTileSelection | CanvasCellSelection;

export type SpriteSheetCanvasState = {
  selectedItems: string[];
  selectedTargets: SpriteSheetSelection[];
};

export type SpriteSheetCanvasStateEvent =
  | { type: "item.toggle"; itemId: string }
  | { type: "item-tile.toggle"; itemId: string; tileId: string }
  | SpriteSheetCanvasEvent;

export const initialSpriteSheetCanvasState: SpriteSheetCanvasState = {
  selectedItems: [],
  selectedTargets: [],
};

export function reduceSpriteSheetCanvas(
  state: SpriteSheetCanvasState,
  event: SpriteSheetCanvasStateEvent,
  items: EditorSpriteSheetItem[],
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
              target.type !== "item-tile" || target.itemId !== event.itemId,
          ),
    };
  }

  if (event.type === "cell.selection.toggled") {
    const target = findCellTarget(event.cellIndex, items);
    if (target.type === "item-tile") {
      return reduceSpriteSheetCanvas(
        state,
        { ...target, type: "item-tile.toggle" },
        items,
      );
    }
    return {
      ...state,
      selectedTargets: toggleSelection(state.selectedTargets, target),
    };
  }

  const target: ItemTileSelection = {
    type: "item-tile",
    itemId: event.itemId,
    tileId: event.tileId,
  };
  const item = items.find((candidate) => candidate.id === target.itemId);
  const itemTargets = item
    ? item.tiles.map((tile) => ({
        type: "item-tile" as const,
        itemId: item.id,
        tileId: tile.id,
      }))
    : [];

  if (item && state.selectedItems.includes(target.itemId)) {
    return {
      selectedItems: state.selectedItems.filter(
        (selectedItem) => selectedItem !== target.itemId,
      ),
      selectedTargets: [
        ...state.selectedTargets.filter(
          (selection) =>
            selection.type !== "item-tile" ||
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
      selectedItems: state.selectedItems.includes(target.itemId)
        ? state.selectedItems
        : [...state.selectedItems, target.itemId],
      selectedTargets: state.selectedTargets.filter(
        (selection) =>
          selection.type !== "item-tile" || selection.itemId !== target.itemId,
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
) {
  const [state, dispatch] = useReducer(
    (current: SpriteSheetCanvasState, event: SpriteSheetCanvasStateEvent) =>
      reduceSpriteSheetCanvas(current, event, items),
    initialSpriteSheetCanvasState,
  );

  return {
    selectedItems: state.selectedItems,
    selectedCells: getSelectedCells(state, items),
    selectedLabels: getSelectedLabels(state, items),
    isTileSelected: (itemId: string, tileId: string) =>
      hasSelection(state.selectedTargets, {
        type: "item-tile",
        itemId,
        tileId,
      }),
    send: (event: SpriteSheetCanvasStateEvent) => dispatch(event),
  };
}

export function getSelectedCells(
  state: SpriteSheetCanvasState,
  items: EditorSpriteSheetItem[],
) {
  return [
    ...state.selectedItems.flatMap(
      (itemId) =>
        items
          .find((item) => item.id === itemId)
          ?.tiles.flatMap((tile) => tile.cells) ?? [],
    ),
    ...state.selectedTargets.flatMap((target) => {
      if (target.type === "canvas-cell") return [target.cellIndex];
      return (
        items
          .find((item) => item.id === target.itemId)
          ?.tiles.find((tile) => tile.id === target.tileId)?.cells ?? []
      );
    }),
  ];
}

export function getSelectedLabels(
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
      const tile = item?.tiles.find(
        (candidate) => candidate.id === target.tileId,
      );
      return `${item?.label ?? target.itemId} / ${tile?.label ?? target.tileId}`;
    }),
  ];
}

function findCellTarget(
  cellIndex: number,
  items: EditorSpriteSheetItem[],
): SpriteSheetSelection {
  let target: SpriteSheetSelection = { type: "canvas-cell", cellIndex };
  for (const item of items) {
    for (const tile of item.tiles) {
      if (tile.cells.includes(cellIndex)) {
        target = { type: "item-tile", itemId: item.id, tileId: tile.id };
      }
    }
  }
  return target;
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

function sameSelection(
  left: SpriteSheetSelection,
  right: SpriteSheetSelection,
) {
  if (left.type === "item-tile" && right.type === "item-tile") {
    return left.itemId === right.itemId && left.tileId === right.tileId;
  }
  if (left.type === "canvas-cell" && right.type === "canvas-cell") {
    return left.cellIndex === right.cellIndex;
  }
  return false;
}
