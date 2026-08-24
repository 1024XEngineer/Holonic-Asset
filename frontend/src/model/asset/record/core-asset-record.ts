import type {
  AssetDetailResponse,
  AssetRecordResponse,
} from "../library/asset.contract";
import { coreAssetApi } from "../library/core-asset.api";
import type { AssetRevision, SceneryLayer } from "../types";
import { projectApi } from "../../project";
import type {
  AssetRecord,
  AssetRecordSaveResult,
  AssetWorkspaceData,
  GetAssetRecordInput,
  SaveAssetRecordInput,
  SceneryCanvasDimensions,
  UISetComponent,
} from "./types";
import {
  toCoreSpriteAssetContent,
  toCoreSpriteAssetWorkspace,
} from "./core-sprite-record";

export async function loadCoreAssetWorkspace(
  input: GetAssetRecordInput,
): Promise<AssetWorkspaceData> {
  const assetId = persistedAssetId(input.assetId);
  const detail = await coreAssetApi.detail(assetId);
  const [project, recordsResponse] = await Promise.all([
    projectApi.detail(input.projectId),
    coreAssetApi.records(assetId),
  ]);
  const workspaceInput = {
    projectId: input.projectId,
    projectName: project.name,
    records: recordsResponse.records,
  };

  switch (detail.type) {
    case "character":
    case "object":
      return toCoreSpriteAssetWorkspace({ ...workspaceInput, detail });
    case "tileSet":
      return toCoreTilesetAssetWorkspace({ ...workspaceInput, detail });
    case "scenery":
      return toCoreSceneryAssetWorkspace({ ...workspaceInput, detail });
    case "uiset":
      return toCoreUISetAssetWorkspace({ ...workspaceInput, detail });
    case "audio":
      throw new Error("Audio assets do not have editable records.");
  }
}

export async function saveCoreAssetRevision(
  input: SaveAssetRecordInput,
): Promise<AssetRecordSaveResult> {
  const assetId = persistedAssetId(input.assetId);
  const expectedVersion = parseVersion(input.version);
  const saved = await coreAssetApi.record({
    assetId,
    ...(expectedVersion ? { expectedVersion } : {}),
    ...(input.description ? { description: input.description } : {}),
    content: toCoreAssetContent(input.record),
  });
  const records = await coreAssetApi.records(assetId);

  return {
    projectId: input.projectId,
    assetId: input.assetId,
    version: `v${saved.version}`,
    history: toHistory(records.records, saved.version),
    record: structuredClone(input.record),
  };
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

  return createWorkspace({
    projectId,
    projectName,
    detail,
    kind: "tileset",
    record,
    records,
  });
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

  return createWorkspace({
    projectId,
    projectName,
    detail,
    kind: "scenery",
    record,
    records,
  });
}

export function toCoreUISetAssetWorkspace({
  projectId,
  projectName,
  detail,
  records,
}: {
  projectId: string;
  projectName: string;
  detail: Extract<AssetDetailResponse, { type: "uiset" }>;
  records: AssetRecordResponse[];
}): AssetWorkspaceData {
  const { width, height } = detail.dimensions;
  const record: AssetRecord = {
    mode: "uiset",
    prompt: detail.description,
    uiset: {
      dimensions: { width, height },
      components: (detail.content?.components ?? []).map((component) => ({
        id: String(component.id),
        label: component.name,
        kind: readUISetComponentKind(component.metadata),
        bounds: {
          x: toPercentage(component.position.x, width),
          y: toPercentage(component.position.y, height),
          width: toPercentage(component.size.width, width),
          height: toPercentage(component.size.height, height),
        },
      })),
    },
  };

  return createWorkspace({
    projectId,
    projectName,
    detail,
    kind: "uiset",
    record,
    records,
  });
}

function createWorkspace({
  projectId,
  projectName,
  detail,
  kind,
  record,
  records,
}: {
  projectId: string;
  projectName: string;
  detail: AssetDetailResponse;
  kind: AssetWorkspaceData["asset"]["kind"];
  record: AssetRecord;
  records: AssetRecordResponse[];
}): AssetWorkspaceData {
  return {
    projectName,
    asset: {
      id: String(detail.assetId),
      projectId,
      kind,
      name: detail.name,
      perspective: detail.perspective,
      version: `v${detail.version}`,
      history: toHistory(records, detail.version),
    },
    record,
  } as AssetWorkspaceData;
}

function toCoreAssetContent(record: AssetRecord) {
  switch (record.mode) {
    case "character":
    case "object":
      return toCoreSpriteAssetContent(record);
    case "tileset":
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
    case "scenery":
      return {
        layers: record.scenery.layers.map((layer, layerIndex) =>
          toCoreSceneryLayer(layer, layerIndex),
        ),
      };
    case "uiset":
      return {
        components: record.uiset.components.map((component, componentIndex) =>
          toCoreUISetComponent(
            component,
            componentIndex,
            record.uiset.dimensions,
          ),
        ),
      };
  }
}

function toCoreSceneryLayer(layer: SceneryLayer, index: number) {
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

function toCoreUISetComponent(
  component: UISetComponent,
  index: number,
  dimensions: { width: number; height: number } | undefined,
) {
  return {
    id: numericId(component.id, index + 1),
    name: component.label,
    size: {
      width: fromPercentage(component.bounds.width, dimensions?.width),
      height: fromPercentage(component.bounds.height, dimensions?.height),
    },
    position: {
      x: fromPercentage(component.bounds.x, dimensions?.width),
      y: fromPercentage(component.bounds.y, dimensions?.height),
    },
    metadata: { kind: component.kind },
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

function readUISetComponentKind(
  metadata: Record<string, unknown> | undefined,
): UISetComponent["kind"] {
  const kind = readString(metadata, "kind");
  return kind === "label" || kind === "button" || kind === "panel"
    ? kind
    : "panel";
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

function toPercentage(value: number, total: number) {
  return total > 0 ? (value / total) * 100 : 0;
}

function fromPercentage(value: number, total: number | undefined) {
  return total && total > 0 ? (value / 100) * total : value;
}

function persistedAssetId(value: string) {
  const assetId = Number(value);
  if (!Number.isSafeInteger(assetId) || assetId <= 0) {
    throw new Error("Asset editor requires a persisted Core API asset.");
  }
  return assetId;
}

function parseVersion(version: string | undefined) {
  const value = Number(version?.replace(/^v/, ""));
  return Number.isSafeInteger(value) && value > 0 ? value : undefined;
}

function numericId(value: string, fallback: number) {
  const id = Number(value);
  return Number.isSafeInteger(id) && id > 0 ? id : fallback;
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
