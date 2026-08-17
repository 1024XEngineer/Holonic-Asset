import type {
  AssetDetailResponse,
  AssetRecordResponse,
} from "../library/asset.contract";
import { coreAssetApi } from "../library/core-asset.api";
import type {
  AssetKind,
  AssetRevision,
  CharacterAnimation,
  CharacterSpriteSheet,
} from "../types";
import { getPerspectiveDirectionLayout } from "../types";
import { projectApi } from "../../project";
import type { AssetRecord, AssetWorkspaceData } from "./types";
import type {
  AssetRecordSaveResult,
  GetAssetRecordInput,
  SaveAssetRecordInput,
} from "./types";

type CoreSpriteAssetKind = Extract<AssetKind, "character" | "object">;

export async function loadCoreSpriteAssetWorkspace(
  input: GetAssetRecordInput,
): Promise<AssetWorkspaceData | undefined> {
  const assetId = Number(input.assetId);
  if (!Number.isSafeInteger(assetId) || assetId <= 0) return undefined;

  const detail = await coreAssetApi.detail(assetId);
  if (detail.type !== "character" && detail.type !== "object") {
    return undefined;
  }

  const [project, recordsResponse] = await Promise.all([
    projectApi.detail(input.projectId),
    coreAssetApi.records(assetId),
  ]);
  return toCoreSpriteAssetWorkspace({
    projectId: input.projectId,
    projectName: project.name,
    detail,
    records: recordsResponse.records,
  });
}

export async function saveCoreSpriteAssetRevision(
  input: SaveAssetRecordInput,
): Promise<AssetRecordSaveResult | undefined> {
  const assetId = Number(input.assetId);
  if (!Number.isSafeInteger(assetId) || assetId <= 0) return undefined;
  if (input.record.mode !== "character" && input.record.mode !== "object") {
    return undefined;
  }

  const expectedVersion = parseVersion(input.version);
  const saved = await coreAssetApi.record({
    assetId,
    ...(expectedVersion ? { expectedVersion } : {}),
    content: toCoreSpriteAssetContent(input.record),
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

export function toCoreSpriteAssetWorkspace({
  projectId,
  projectName,
  detail,
  records,
}: {
  projectId: string;
  projectName: string;
  detail: AssetDetailResponse;
  records: AssetRecordResponse[];
}): AssetWorkspaceData {
  if (detail.type !== "character" && detail.type !== "object") {
    throw new Error("Core sprite records require a Character or Object asset.");
  }

  const kind = detail.type;
  const sprite = {
    prototype: toPrototype(detail),
    animations: toAnimations(detail),
    nodePositions: readNodePositions(detail.content?.metadata),
  };
  const record: AssetRecord =
    kind === "character"
      ? { mode: kind, prompt: detail.description, character: sprite }
      : { mode: kind, prompt: detail.description, object: sprite };

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

function toPrototype(
  detail: Extract<AssetDetailResponse, { type: CoreSpriteAssetKind }>,
): CharacterSpriteSheet {
  const frameUrls = readURLs(detail.content?.prototype);
  const layout = getPerspectiveDirectionLayout(detail.perspective);
  return {
    format: "png-sprite-sheet",
    imageUrl: frameUrls[0] ?? "",
    ...(frameUrls.length > 1 ? { frameUrls } : {}),
    frameWidth: detail.dimensions.width,
    frameHeight: detail.dimensions.height,
    columns: layout.columns,
    rows: layout.rows,
  };
}

function toAnimations(
  detail: Extract<AssetDetailResponse, { type: CoreSpriteAssetKind }>,
): CharacterAnimation[] {
  return (detail.content?.animations ?? []).map((animation) => {
    const frameUrls = readURLs(animation.frames);
    return {
      kind: "clip",
      id: String(animation.id),
      label: animation.name,
      frameCount: frameUrls.length,
      ...(frameUrls.length > 0
        ? {
            spriteSheet: {
              format: "png-sprite-sheet",
              imageUrl: frameUrls[0]!,
              frameUrls,
              frameWidth: detail.dimensions.width,
              frameHeight: detail.dimensions.height,
              columns: frameUrls.length,
              rows: 1,
            },
          }
        : {}),
    };
  });
}

function readURLs(resources: Array<{ url?: string }> | undefined) {
  return (resources ?? []).flatMap((resource) =>
    resource.url ? [resource.url] : [],
  );
}

function toCoreSpriteAssetContent(record: AssetRecord) {
  if (record.mode !== "character" && record.mode !== "object") {
    throw new Error("Core sprite records require a Character or Object asset.");
  }
  const sprite = record.mode === "character" ? record.character : record.object;
  const prototypeURLs = sprite.prototype.frameUrls ?? [
    sprite.prototype.imageUrl,
  ];
  return {
    directionCount: prototypeURLs.filter(Boolean).length,
    prototype: prototypeURLs.filter(Boolean).map((url, index) => ({
      id: index + 1,
      url,
    })),
    animations: (sprite.animations ?? []).map((animation, animationIndex) => ({
      id: numericId(animation.id, animationIndex + 1),
      name: animation.label,
      frames: (animation.spriteSheet?.frameUrls ?? []).map(
        (url, frameIndex) => ({
          id: frameIndex + 1,
          url,
        }),
      ),
    })),
    metadata: { nodePositions: structuredClone(sprite.nodePositions) },
  };
}

function readNodePositions(metadata: Record<string, unknown> | undefined) {
  const value = metadata?.nodePositions;
  if (!value || typeof value !== "object" || Array.isArray(value)) return {};
  return Object.fromEntries(
    Object.entries(value).flatMap(([id, position]) => {
      if (
        !position ||
        typeof position !== "object" ||
        Array.isArray(position)
      ) {
        return [];
      }
      const { x, y } = position as { x?: unknown; y?: unknown };
      return typeof x === "number" && typeof y === "number"
        ? [[id, { x, y }]]
        : [];
    }),
  );
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
