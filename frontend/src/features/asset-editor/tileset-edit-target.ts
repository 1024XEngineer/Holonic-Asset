import type { ItemTile, TilesetItem } from "@/model";

export const MAX_TILESET_EDIT_TARGETS = 256;

export type ResolvedTilesetEditTarget =
  | {
      kind: "item";
      itemId: string;
      label: string;
      position: ItemTile;
      positions: ItemTile[];
    }
  | {
      kind: "tiles";
      label: string;
      positions: ItemTile[];
    };

export type TilesetEditTargetResolution = {
  target: ResolvedTilesetEditTarget | null;
  error: "missing" | "too-many" | "multiple-items" | null;
};

export function resolveTilesetEditTarget({
  selectedCellIndexes,
  items,
  gridSize,
}: {
  selectedCellIndexes: readonly number[];
  items: readonly TilesetItem[];
  gridSize: number;
}): TilesetEditTargetResolution {
  if (!Number.isInteger(gridSize) || gridSize <= 0) {
    return missingTarget();
  }

  const occupied = createOccupiedCellIndex(items, gridSize);
  const selected = [...new Set(selectedCellIndexes)]
    .filter(
      (cellIndex) => Number.isInteger(cellIndex) && occupied.has(cellIndex),
    )
    .sort((left, right) => left - right)
    .map((cellIndex) => occupied.get(cellIndex)!);
  if (selected.length === 0) return missingTarget();
  if (selected.length > MAX_TILESET_EDIT_TARGETS) {
    return {
      target: null,
      error: "too-many",
    };
  }

  if (new Set(selected.map(({ itemId }) => itemId)).size > 1) {
    return {
      target: null,
      error: "multiple-items",
    };
  }

  const selectedPositionKeys = new Set(
    selected.map(({ position }) => positionKey(position)),
  );
  const completeItems = items.flatMap((item) => {
    const positions = uniqueValidPositions(item.tiles, gridSize);
    return positions.length > 0 &&
      positions.length === selectedPositionKeys.size &&
      positions.every((position) =>
        selectedPositionKeys.has(positionKey(position)),
      )
      ? [{ item, positions }]
      : [];
  });
  if (completeItems.length === 1) {
    const { item, positions } = completeItems[0]!;
    return {
      target: {
        kind: "item",
        itemId: item.id,
        label: item.label,
        position: positions[0]!,
        positions,
      },
      error: null,
    };
  }

  return {
    target: {
      kind: "tiles",
      label: selected.map(({ label }) => label).join(", "),
      positions: selected.map(({ position }) => position),
    },
    error: null,
  };
}

function createOccupiedCellIndex(
  items: readonly TilesetItem[],
  gridSize: number,
) {
  const occupied = new Map<
    number,
    { position: ItemTile; label: string; itemId: string }
  >();
  for (const item of items) {
    item.tiles.forEach((position, itemCellIndex) => {
      const cellIndex = toCellIndex(position, gridSize);
      if (cellIndex === undefined || occupied.has(cellIndex)) return;
      occupied.set(cellIndex, {
        position: [position[0], position[1]],
        label: `${item.label} / Tile ${itemCellIndex + 1}`,
        itemId: item.id,
      });
    });
  }
  return occupied;
}

function uniqueValidPositions(
  positions: readonly ItemTile[],
  gridSize: number,
) {
  const unique = new Map<string, ItemTile>();
  for (const position of positions) {
    if (toCellIndex(position, gridSize) === undefined) continue;
    unique.set(positionKey(position), [position[0], position[1]]);
  }
  return [...unique.values()];
}

function toCellIndex([x, y]: ItemTile, gridSize: number) {
  if (
    !Number.isInteger(x) ||
    !Number.isInteger(y) ||
    x < 0 ||
    y < 0 ||
    x >= gridSize ||
    y >= gridSize
  ) {
    return undefined;
  }
  return y * gridSize + x;
}

function positionKey([x, y]: ItemTile) {
  return `${x}:${y}`;
}

function missingTarget(): TilesetEditTargetResolution {
  return {
    target: null,
    error: "missing",
  };
}
