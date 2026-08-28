import type {
  AssetDetailResponse,
  AssetRecordResponse,
  CharacterAssetContent,
  BackendSpriteContent,
  BackendSpriteContentPatch,
} from "../library/asset.contract";
import { coreAssetApi } from "../library/core-asset.api";
import { mergeAssetContentPatch } from "../library/merge-asset-content";
import type {
  AssetKind,
  CharacterAnimation,
  CharacterSpriteSheet,
} from "../types";
import { getPerspectiveDirectionLayout } from "../types";
import { projectApi } from "../../project";
import type {
  AssetRecord,
  AssetWorkspaceData,
  AssetRecordSaveResult,
  GetAssetRecordInput,
  SaveAssetRecordInput,
} from "../record/types";
import {
  createAssetSnapshot,
  toAssetHistory,
} from "../record/record-history.mapper";

type CoreSpriteAssetKind = Extract<AssetKind, "character" | "object">;

export async function loadSpriteAssetContent(
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
  return toSpriteAssetContent({
    projectId: input.projectId,
    projectName: project.name,
    detail,
    records: recordsResponse.records,
  });
}

export async function saveSpriteRevision(
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
    ...(input.description ? { description: input.description } : {}),
    content: toBackendSpriteContent(input.record),
  });
  const records = await coreAssetApi.records(assetId);
  return {
    projectId: input.projectId,
    assetId: input.assetId,
    version: `v${saved.version}`,
    history: toAssetHistory(records.records, saved.version),
    record: structuredClone(input.record),
  };
}

export function toSpriteAssetContent({
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
    nodePositions: {},
  };
  const record: AssetRecord =
    kind === "character"
      ? { mode: kind, prompt: detail.description, character: sprite }
      : { mode: kind, prompt: detail.description, object: sprite };

  return createAssetSnapshot({
    projectId,
    projectName,
    detail,
    kind,
    record,
    records,
  });
}

export function toSpriteContentCandidate(
  record: AssetRecord,
  perspective: AssetWorkspaceData["asset"]["perspective"],
  patch: BackendSpriteContentPatch,
): AssetRecord {
  if (record.mode !== "character" && record.mode !== "object") {
    throw new Error(
      "Core sprite candidates require a Character or Object asset.",
    );
  }
  const currentSprite =
    record.mode === "character" ? record.character : record.object;
  const content = mergeAssetContentPatch(toBackendSpriteContent(record), patch);
  const sprite = {
    prototype: toPrototypeFromContent(
      content.prototype,
      perspective,
      currentSprite.prototype.frameWidth,
      currentSprite.prototype.frameHeight,
    ),
    animations: toAnimationsFromContent(
      content.animations,
      currentSprite.prototype.frameWidth,
      currentSprite.prototype.frameHeight,
    ),
    nodePositions: {},
  };
  return record.mode === "character"
    ? { ...record, character: sprite }
    : { ...record, object: sprite };
}

function toPrototype(
  detail: Extract<AssetDetailResponse, { type: CoreSpriteAssetKind }>,
): CharacterSpriteSheet {
  return toPrototypeFromContent(
    detail.content?.prototype,
    detail.perspective,
    detail.dimensions.width,
    detail.dimensions.height,
  );
}

function toPrototypeFromContent(
  prototype: CharacterAssetContent["prototype"] | undefined,
  perspective: AssetWorkspaceData["asset"]["perspective"],
  frameWidth: number,
  frameHeight: number,
): CharacterSpriteSheet {
  const frameUrls = readURLs(prototype);
  const layout = getPerspectiveDirectionLayout(perspective);
  return {
    format: "png-sprite-sheet",
    imageUrl: frameUrls[0] ?? "",
    ...(frameUrls.length > 1 ? { frameUrls } : {}),
    frameWidth,
    frameHeight,
    columns: layout.columns,
    rows: layout.rows,
  };
}

function toAnimations(
  detail: Extract<AssetDetailResponse, { type: CoreSpriteAssetKind }>,
): CharacterAnimation[] {
  return toAnimationsFromContent(
    detail.content?.animations,
    detail.dimensions.width,
    detail.dimensions.height,
  );
}

function toAnimationsFromContent(
  animations: CharacterAssetContent["animations"] | undefined,
  frameWidth: number,
  frameHeight: number,
): CharacterAnimation[] {
  return (animations ?? []).map((animation) => {
    const frames = (animation.frames ?? []).flatMap((frame) =>
      frame.url ? [{ url: frame.url, duration: frame.duration }] : [],
    );
    const frameUrls = frames.map((frame) => frame.url);
    return {
      kind: "clip",
      id: String(animation.id),
      ...(animation.groupId ? { groupId: String(animation.groupId) } : {}),
      label: animation.name,
      frameCount: frameUrls.length,
      ...(frames.some((frame) => frame.duration !== undefined)
        ? { frameDurations: frames.map((frame) => frame.duration) }
        : {}),
      ...(animation.generation
        ? { generation: structuredClone(animation.generation) }
        : {}),
      ...(frameUrls.length > 0
        ? {
            spriteSheet: {
              format: "png-sprite-sheet",
              imageUrl: frameUrls[0]!,
              frameUrls,
              frameWidth,
              frameHeight,
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

export function toBackendSpriteContent(
  record: AssetRecord,
): BackendSpriteContent {
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
      ...(animation.groupId
        ? { groupId: numericId(animation.groupId, 0) }
        : {}),
      name: animation.label,
      frames: (animation.spriteSheet?.frameUrls ?? []).map(
        (url, frameIndex) => ({
          id: frameIndex + 1,
          url,
          ...(animation.frameDurations?.[frameIndex] !== undefined
            ? { duration: animation.frameDurations[frameIndex] }
            : {}),
        }),
      ),
      ...(animation.generation
        ? { generation: structuredClone(animation.generation) }
        : {}),
    })),
  };
}

function parseVersion(version: string | undefined) {
  const value = Number(version?.replace(/^v/, ""));
  return Number.isSafeInteger(value) && value > 0 ? value : undefined;
}

function numericId(value: string, fallback: number) {
  const id = Number(value);
  return Number.isSafeInteger(id) && id > 0 ? id : fallback;
}
