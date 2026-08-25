import type {
  AssetDetailResponse,
  AssetRecordResponse,
  TileSetAssetContent,
} from "../library/asset.contract";
import type { AssetRecord, AssetWorkspaceData, TilesetItem } from "./types";
import { createCoreAssetWorkspace } from "./core-asset-workspace";

export function toCoreTilesetAssetWorkspace({
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

  return createCoreAssetWorkspace({
    projectId,
    projectName,
    detail,
    kind: "tileset",
    record,
    records,
  });
}

export function toCoreTilesetCandidateRecord(
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

  return {
    ...record,
    tileset: {
      ...record.tileset,
      items: record.tileset.items.map((item) => ({
        ...item,
        tileUrls: item.tiles.map(
          ([x, y], tileIndex) =>
            patchedURLs.get(tilesetPositionKey(x, y)) ??
            item.tileUrls?.[tileIndex],
        ),
      })),
    },
  };
}

export function getCoreTilesetCandidateItemIds(
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
  return items
    .filter((item) =>
      item.tiles.some(([x, y]) =>
        changedPositions.has(tilesetPositionKey(x, y)),
      ),
    )
    .map((item) => item.id);
}

export function toCoreTilesetAssetContent(
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
