import type {
  AssetDetailResponse,
  AssetRecordResponse,
  TileSetAssetContent,
} from "../library/asset.contract";
import type {
  AssetRecord,
  AssetWorkspaceData,
  TilesetItem,
} from "../record/types";
import { createAssetSnapshot } from "../record/record-history.mapper";

export function toTilesetAssetContent({
  projectId,
  projectName,
  detail,
  records,
}: {
  projectId: string;
  projectName: string;
  detail: Extract<AssetDetailResponse, { type: "tileSet" }>;
  records: AssetRecordResponse[];
}): AssetWorkspaceData {
  const record: AssetRecord = {
    mode: "tileset",
    prompt: detail.description,
    tileset: {
      gridSize: detail.dimensions.tileAmount.columns,
      items: (detail.content?.items ?? []).map((item, itemIndex) => ({
        id: `${itemIndex + 1}:${item.name}`,
        label: item.name,
        tiles: (item.tiles ?? []).map(
          (tile) => [tile.position.x, tile.position.y] as [number, number],
        ),
        tileUrls: (item.tiles ?? []).map((tile) => tile.url),
      })),
    },
  };

  return createAssetSnapshot({
    projectId,
    projectName,
    detail,
    kind: "tileset",
    record,
    records,
  });
}

export function toTilesetContentCandidate(
  record: AssetRecord,
  patch: TileSetAssetContent,
): AssetRecord {
  if (record.mode !== "tileset") {
    throw new Error("Core Tileset candidates require a Tileset asset.");
  }

  const patchedURLs = new Map<string, string>();
  for (const item of patch.items ?? []) {
    for (const tile of item.tiles ?? []) {
      if (!tile.url?.trim()) continue;
      patchedURLs.set(
        tilesetPositionKey(tile.position.x, tile.position.y),
        tile.url,
      );
    }
  }

  const occupiedPositions = new Set(
    record.tileset.items.flatMap((item) =>
      item.tiles.map(([x, y]) => tilesetPositionKey(x, y)),
    ),
  );
  const addedItems = (patch.items ?? []).flatMap((item, itemIndex) => {
    const tiles = (item.tiles ?? [])
      .filter(
        ({ position }) =>
          Number.isInteger(position.x) &&
          Number.isInteger(position.y) &&
          position.x >= 0 &&
          position.y >= 0 &&
          position.x < record.tileset.gridSize &&
          position.y < record.tileset.gridSize,
      )
      .map(({ position }) => [position.x, position.y] as [number, number]);
    if (
      tiles.length === 0 ||
      tiles.some(([x, y]) => occupiedPositions.has(tilesetPositionKey(x, y))) ||
      !tiles.some(([x, y]) => patchedURLs.has(tilesetPositionKey(x, y)))
    ) {
      return [];
    }
    return [
      {
        id: candidateTilesetItemId(item.name, itemIndex),
        label: item.name,
        tiles,
        tileUrls: tiles.map(([x, y]) =>
          patchedURLs.get(tilesetPositionKey(x, y)),
        ),
      },
    ];
  });

  return {
    ...record,
    tileset: {
      ...record.tileset,
      items: [
        ...record.tileset.items.map((item) => ({
          ...item,
          tileUrls: item.tiles.map(
            ([x, y], tileIndex) =>
              patchedURLs.get(tilesetPositionKey(x, y)) ??
              item.tileUrls?.[tileIndex],
          ),
        })),
        ...addedItems,
      ],
    },
  };
}

export function getTilesetCandidateItemIds(
  patch: TileSetAssetContent | undefined,
  items: readonly TilesetItem[],
  gridSize: number,
) {
  if (!patch || !Number.isInteger(gridSize) || gridSize <= 0) return [];
  const occupied = new Set(
    items.flatMap((item) =>
      item.tiles.map(([x, y]) => tilesetPositionKey(x, y)),
    ),
  );
  const changedPositions = new Set<string>();
  for (const item of patch.items ?? []) {
    for (const tile of item.tiles ?? []) {
      const { x, y } = tile.position;
      if (
        !tile.url?.trim() ||
        !Number.isInteger(x) ||
        !Number.isInteger(y) ||
        x < 0 ||
        y < 0 ||
        x >= gridSize ||
        y >= gridSize ||
        !occupied.has(tilesetPositionKey(x, y))
      ) {
        continue;
      }
      changedPositions.add(tilesetPositionKey(x, y));
    }
  }
  const changedItemIds = items
    .filter((item) =>
      item.tiles.some(([x, y]) =>
        changedPositions.has(tilesetPositionKey(x, y)),
      ),
    )
    .map((item) => item.id);
  const occupiedPositions = new Set(
    items.flatMap((item) =>
      item.tiles.map(([x, y]) => tilesetPositionKey(x, y)),
    ),
  );
  for (const [itemIndex, item] of (patch.items ?? []).entries()) {
    const tiles = item.tiles ?? [];
    const positions = tiles.map(({ position }) => position);
    if (
      positions.length > 0 &&
      positions.every(
        ({ x, y }) =>
          Number.isInteger(x) &&
          Number.isInteger(y) &&
          x >= 0 &&
          y >= 0 &&
          x < gridSize &&
          y < gridSize &&
          !occupiedPositions.has(tilesetPositionKey(x, y)),
      ) &&
      positions.some(({ x, y }) =>
        tiles.some(
          (tile) =>
            tile.position.x === x &&
            tile.position.y === y &&
            Boolean(tile.url?.trim()),
        ),
      )
    ) {
      changedItemIds.push(candidateTilesetItemId(item.name, itemIndex));
    }
  }
  return changedItemIds;
}

export function toBackendTilesetContent(
  record: Extract<AssetRecord, { mode: "tileset" }>,
) {
  return {
    items: record.tileset.items.map((item) => ({
      name: item.label,
      tiles: item.tiles.map(([x, y], tileIndex) => ({
        ...(item.tileUrls?.[tileIndex]
          ? { url: item.tileUrls[tileIndex] }
          : {}),
        position: { x, y },
      })),
    })),
  };
}

function tilesetPositionKey(x: number, y: number) {
  return `${x}:${y}`;
}

function candidateTilesetItemId(name: string, index: number) {
  return `candidate:${index}:${name.trim() || "item"}`;
}
