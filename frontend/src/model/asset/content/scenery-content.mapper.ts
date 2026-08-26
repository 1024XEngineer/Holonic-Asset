import type {
  AssetDetailResponse,
  AssetRecordResponse,
} from "../library/asset.contract";
import type { SceneryLayer } from "../types";
import type {
  AssetRecord,
  AssetWorkspaceData,
  SceneryCanvasDimensions,
} from "../record/types";
import { createAssetSnapshot } from "../record/record-history.mapper";

export function toSceneryAssetContent({
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
        detail: readString(layer.metadata, "detail") ?? layer.name,
        imageUrl: layer.resource,
        blendMode: readBlendMode(layer.metadata),
        position: layer.position,
        transform: toSceneryTransform(layer.transform),
        ...(layer.visible === undefined ? {} : { visible: layer.visible }),
        ...(layer.opacity === undefined ? {} : { opacity: layer.opacity }),
        ...(layer.zIndex === undefined ? {} : { zIndex: layer.zIndex }),
      })),
    },
  };

  return createAssetSnapshot({
    projectId,
    projectName,
    detail,
    kind: "scenery",
    record,
    records,
  });
}

export function toBackendSceneryContent(
  record: Extract<AssetRecord, { mode: "scenery" }>,
) {
  return {
    layers: record.scenery.layers.map((layer, layerIndex) =>
      toBackendSceneryLayer(layer, layerIndex),
    ),
  };
}

function toBackendSceneryLayer(layer: SceneryLayer, index: number) {
  return {
    id: numericId(layer.id, index + 1),
    name: layer.label,
    resource: layer.imageUrl,
    position: layer.position ?? { x: 0, y: 0 },
    ...(layer.transform ? { transform: layer.transform } : {}),
    ...(layer.visible === undefined ? {} : { visible: layer.visible }),
    ...(layer.opacity === undefined ? {} : { opacity: layer.opacity }),
    ...(layer.zIndex === undefined ? {} : { zIndex: layer.zIndex }),
    metadata: { detail: layer.detail, blendMode: layer.blendMode },
  };
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

function readBlendMode(metadata: Record<string, unknown> | undefined) {
  return readString(metadata, "blendMode") === "multiply"
    ? "multiply"
    : "normal";
}

function readString(value: Record<string, unknown> | undefined, key: string) {
  const field = value?.[key];
  return typeof field === "string" ? field : undefined;
}

function toFiniteNumber(value: unknown) {
  return typeof value === "number" && Number.isFinite(value)
    ? value
    : undefined;
}

function numericId(value: string, fallback: number) {
  const id = Number(value);
  return Number.isSafeInteger(id) && id > 0 ? id : fallback;
}
