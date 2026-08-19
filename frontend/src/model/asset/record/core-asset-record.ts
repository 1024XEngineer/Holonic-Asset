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
  SceneryCanvasDimensions,
} from "./types";
import type { GetAssetRecordInput } from "./types";
import { toCoreSpriteAssetWorkspace } from "./core-sprite-record";

type CoreWorkspaceAssetKind = Extract<
  AssetDetailResponse["type"],
  "character" | "object" | "scenery" | "tileSet"
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

  if (detail.type === "scenery") {
    return toCoreSceneryAssetWorkspace({
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

export function toCoreSceneryAssetWorkspace({
  projectId,
  projectName,
  detail,
  records,
}: {
  projectId: string;
  projectName: string;
  detail: Extract<AssetDetailResponse, { type: "scenery" }>;
  records: AssetRecordResponse[];
}): AssetWorkspaceData {
  const record: AssetRecord = {
    mode: "scenery",
    prompt: detail.description,
    scenery: {
      dimensions: toSceneryCanvasDimensions(detail.dimensions),
      layers: (detail.content?.layers ?? []).map((layer) => ({
        id: String(layer.id),
        label: layer.name,
        detail: layer.name,
        imageUrl: layer.resource,
        blendMode: "normal",
        position: layer.position,
        transform: toSceneryTransform(layer.transform),
        ...(layer.visible === undefined ? {} : { visible: layer.visible }),
        ...(layer.opacity === undefined ? {} : { opacity: layer.opacity }),
        ...(layer.zIndex === undefined ? {} : { zIndex: layer.zIndex }),
      })),
    },
  };

  return {
    projectName,
    asset: {
      id: String(detail.assetId),
      projectId,
      kind: "scenery",
      name: detail.name,
      perspective: detail.perspective,
      version: `v${detail.version}`,
      history: toHistory(records, detail.version),
    },
    record,
  } as AssetWorkspaceData;
}

function toSceneryCanvasDimensions(
  dimensions: Extract<AssetDetailResponse, { type: "scenery" }>["dimensions"],
): SceneryCanvasDimensions {
  return { width: dimensions.width, height: dimensions.height };
}

function toSceneryTransform(value: unknown) {
  if (!value || typeof value !== "object") return undefined;
  const transform = value as Record<string, unknown>;
  const scale = transform.scale;
  if (!scale || typeof scale !== "object") return undefined;
  const scaleRecord = scale as Record<string, unknown>;
  const x = toFiniteNumber(scaleRecord.x);
  const y = toFiniteNumber(scaleRecord.y);
  const rotation = toFiniteNumber(transform.rotation);
  if (x === undefined || y === undefined || rotation === undefined) {
    return undefined;
  }
  return { scale: { x, y }, rotation };
}

function toFiniteNumber(value: unknown) {
  return typeof value === "number" && Number.isFinite(value)
    ? value
    : undefined;
}

function isCoreWorkspaceAsset(
  detail: AssetDetailResponse,
): detail is Extract<AssetDetailResponse, { type: CoreWorkspaceAssetKind }> {
  return (
    detail.type === "character" ||
    detail.type === "object" ||
    detail.type === "scenery" ||
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
