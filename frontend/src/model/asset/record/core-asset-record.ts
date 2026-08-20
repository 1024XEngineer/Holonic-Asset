import type {
  AssetDetailResponse,
  AssetRecordResponse,
} from "../library/asset.contract";
import { coreAssetApi } from "../library/core-asset.api";
import type { AssetRevision } from "../types";
import { projectApi } from "../../project";
import type {
  AssetRecord,
  AssetWorkspaceData,
  GetAssetRecordInput,
} from "./types";
import { toCoreSpriteAssetWorkspace } from "./core-sprite-record";

type CoreWorkspaceAssetKind = Extract<
  AssetDetailResponse["type"],
  "character" | "object" | "tileSet"
>;

export async function loadCoreAssetWorkspace(
  input: GetAssetRecordInput,
): Promise<AssetWorkspaceData | undefined> {
  const assetId = Number(input.assetId);
  if (!Number.isSafeInteger(assetId) || assetId <= 0) return undefined;

  const detail = await coreAssetApi.detail(assetId);
  if (!isCoreWorkspaceAsset(detail)) return undefined;

  const [project, recordsResponse] = await Promise.all([
    projectApi.detail(input.projectId),
    coreAssetApi.records(assetId),
  ]);

  if (detail.type === "tileSet") {
    return toCoreTilesetAssetWorkspace({
      projectId: input.projectId,
      projectName: project.name,
      detail,
      records: recordsResponse.records,
    });
  }

  return toCoreSpriteAssetWorkspace({
    projectId: input.projectId,
    projectName: project.name,
    detail,
    records: recordsResponse.records,
  });
}

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

  return {
    projectName,
    asset: {
      id: String(detail.assetId),
      projectId,
      kind: "tileset",
      name: detail.name,
      perspective: detail.perspective,
      version: `v${detail.version}`,
      history: toHistory(records, detail.version),
    },
    record,
  } as AssetWorkspaceData;
}

function isCoreWorkspaceAsset(
  detail: AssetDetailResponse,
): detail is Extract<AssetDetailResponse, { type: CoreWorkspaceAssetKind }> {
  return (
    detail.type === "character" ||
    detail.type === "object" ||
    detail.type === "tileSet"
  );
}

function toHistory(records: AssetRecordResponse[], currentVersion: number) {
  return records.map(
    (record): AssetRevision => ({
      id: String(record.recordId),
      version: `v${record.version}`,
      description: record.description,
      savedAt: record.createdAt,
      status: "ready",
      isCurrent: record.version === currentVersion,
    }),
  );
}
