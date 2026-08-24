import type {
  AssetDetailResponse,
  AssetRecordResponse,
} from "../library/asset.contract";
import type { AssetRecord, AssetWorkspaceData } from "./types";
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
