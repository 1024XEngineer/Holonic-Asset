import { uploadFile } from "@/model/upload";

import {
  assetCanvasSizeDimensionsSchema,
  defaultAssetCanvasSize,
  getDefaultAssetCanvasSize,
} from "../../asset/library/asset-canvas-size";
import type { CreatableAssetKind } from "../../asset";
import { coreAssetApi } from "../../asset/library/core-asset.api";
import { coreGenerationApi } from "./core-generation.api";
import type {
  CreateGenerationRequest,
  GenerationRunListItemResponse,
} from "./generation.contract";
import type { CreationRequest, GenerationInput, GenerationRun } from "./types";

type GenerationRequestMetadata = Pick<
  CreationRequest,
  "kind" | "name" | "prompt"
> &
  Partial<Pick<CreationRequest, "canvasSize" | "perspective">> & {
    assetId?: string;
  };

export type { GenerationInput } from "./types";

export { coreGenerationApi } from "./core-generation.api";
export type {
  CancelGenerationResponse,
  CreateGenerationRequest,
  CreateGenerationResponse,
  DeleteGenerationResponse,
  GenerationRunListItemResponse,
  GenerationRunResponse,
  GenerationTaskStatus,
  GenerationTaskType,
  ListGenerationRunsQuery,
  ListGenerationRunsResponse,
  RetryGenerationResponse,
} from "./generation.contract";

export type GenerationApi = {
  listRuns: (projectId: string, assetId?: string) => Promise<GenerationRun[]>;
  enqueue: (input: GenerationInput) => Promise<GenerationRun>;
};

export const generationApi: GenerationApi = {
  listRuns: async (projectId, assetId) => {
    const response = await coreGenerationApi.list(coreProjectId(projectId), {
      status: "active",
      ...(assetId ? { assetId: coreAssetId(assetId) } : {}),
    });
    const storedMetadata = readGenerationRequests();
    const runs = await Promise.all(
      response.items.map(async (item) => {
        const request =
          generationRequests.get(runKey(projectId, item.id)) ??
          storedMetadata[runKey(projectId, item.id)];
        return toGenerationRun(
          item,
          request,
          await resolveSpriteAssetKind(item, request?.kind),
        );
      }),
    );
    return runs.flatMap((run) => (run ? [run] : []));
  },
  enqueue: async (input) => {
    const request = await toCreateGenerationRequest(input.request);
    const response = await coreGenerationApi.create(
      coreProjectId(input.projectId),
      request,
    );
    const metadata = toGenerationRequestMetadata(input.request);
    const run: GenerationRun = {
      ...metadata,
      canvasSize: metadata.canvasSize ?? defaultAssetCanvasSize,
      id: String(response.generationRunId),
      projectId: input.projectId,
      status: "pending",
    };
    generationRequests.set(runKey(input.projectId, run.id), metadata);
    writeGenerationRequest(runKey(input.projectId, run.id), metadata);
    return run;
  },
};

const generationRequests = new Map<string, GenerationRequestMetadata>();
const animationAssetKinds = new Map<number, "character" | "object">();
const generationRequestsStorageKey = "holonic-generation-requests";

export async function toCreateGenerationRequest(
  request: CreationRequest,
): Promise<CreateGenerationRequest> {
  if (request.kind === "scenery") {
    return {
      kind: "generate_scenery",
      creative_brief: request.prompt,
      parameters: {
        asset_name: request.name,
        dimensions:
          request.dimensions ??
          assetCanvasSizeDimensionsSchema.parse(request.canvasSize),
        creating_reference: await resolveCreatingReference(
          request.creatingReference,
        ),
      },
    };
  }

  if (request.kind === "tileset") {
    if (!request.tiles || request.tiles.length === 0) {
      throw new Error("At least one tileset item is required.");
    }

    return {
      kind: "generate_tileset",
      creative_brief: request.prompt,
      parameters: {
        asset_name: request.name,
        dimensions: {
          tileSize: assetCanvasSizeDimensionsSchema.parse(request.canvasSize),
          tileAmount: { columns: 16, rows: 16 },
        },
        items: request.tiles.map((item) => ({
          name: item.name.trim(),
          description: item.description.trim(),
          shape: item.shape,
        })),
      },
    };
  }

  if (request.kind !== "character" && request.kind !== "object") {
    throw new Error(
      "Core API creation currently supports Character, Object, Scenery, and Tileset assets only.",
    );
  }
  if (!request.perspective) {
    throw new Error("Perspective is required for Character and Object assets.");
  }

  return {
    kind:
      request.kind === "character"
        ? "generate_character_prototype"
        : "generate_object_prototype",
    creative_brief: request.prompt,
    parameters: {
      asset_name: request.name,
      dimensions: assetCanvasSizeDimensionsSchema.parse(request.canvasSize),
      perspective: request.perspective,
      tags: (request.tags ?? []).map(({ name, description, color }) => ({
        name,
        description,
        color,
      })),
      creating_reference: await resolveCreatingReference(
        request.creatingReference,
      ),
    },
  };
}

function toGenerationRun(
  item: GenerationRunListItemResponse,
  request: GenerationRequestMetadata | undefined,
  resolvedAnimationKind: "character" | "object" | undefined,
): GenerationRun | undefined {
  const kind = generationKindToAssetKind(
    item.kind,
    request?.kind,
    resolvedAnimationKind,
  );
  if (!kind || !isVisibleGenerationStatus(item.status)) return undefined;

  return {
    kind,
    name: request?.name ?? `New ${kind}`,
    prompt: request?.prompt ?? "",
    canvasSize: request?.canvasSize ?? getDefaultAssetCanvasSize(kind),
    perspective: request?.perspective,
    id: String(item.id),
    projectId: String(item.projectId),
    assetId: item.assetId === undefined ? undefined : String(item.assetId),
    status: item.status,
  };
}

function generationKindToAssetKind(
  kind: GenerationRunListItemResponse["kind"],
  requestedKind?: CreatableAssetKind,
  resolvedAnimationKind?: "character" | "object",
) {
  switch (kind) {
    case "generate_scenery":
      return "scenery";
    case "generate_character_prototype":
    case "edit_character_prototype":
      return "character";
    case "generate_object_prototype":
    case "edit_object_prototype":
      return "object";
    case "generate_tileset":
    case "add_tileset_item":
    case "edit_tileset_item":
    case "edit_tiles":
      return "tileset";
    case "generate_animation":
    case "edit_animation":
      if (requestedKind === "character" || requestedKind === "object") {
        return requestedKind;
      }
      return resolvedAnimationKind ?? "character";
    case "edit_frames":
      return resolvedAnimationKind ?? "character";
    default:
      return undefined;
  }
}

async function resolveSpriteAssetKind(
  item: GenerationRunListItemResponse,
  requestedKind: CreatableAssetKind | undefined,
): Promise<"character" | "object" | undefined> {
  if (requestedKind === "character" || requestedKind === "object") {
    return requestedKind;
  }
  if (
    (item.kind !== "generate_animation" &&
      item.kind !== "edit_animation" &&
      item.kind !== "edit_frames") ||
    item.assetId === undefined
  ) {
    return undefined;
  }
  const cachedKind = animationAssetKinds.get(item.assetId);
  if (cachedKind) return cachedKind;
  try {
    const asset = await coreAssetApi.detail(item.assetId);
    if (asset.type !== "character" && asset.type !== "object") return undefined;
    animationAssetKinds.set(item.assetId, asset.type);
    return asset.type;
  } catch {
    return undefined;
  }
}

function isVisibleGenerationStatus(
  status: GenerationRunListItemResponse["status"],
): status is GenerationRun["status"] {
  return (
    status === "pending" ||
    status === "processing" ||
    status === "awaiting_application" ||
    status === "failed"
  );
}

async function resolveCreatingReference(creatingReference: unknown) {
  if (creatingReference === undefined) return "";
  if (typeof creatingReference === "string") return creatingReference.trim();
  if (typeof File !== "undefined" && creatingReference instanceof File) {
    return (await uploadFile(creatingReference)).objectKey;
  }
  throw new Error("Creating reference must be an image file or URL.");
}

function toGenerationRequestMetadata(
  request: CreationRequest,
): GenerationRequestMetadata {
  const { kind, name, prompt, canvasSize, perspective } = request;
  return { kind, name, prompt, canvasSize, perspective };
}

function runKey(projectId: string, runId: string | number) {
  return `${projectId}:${runId}`;
}

export function rememberGenerationRunMetadata(
  projectId: string,
  runId: string | number,
  metadata: GenerationRequestMetadata,
) {
  const key = runKey(projectId, runId);
  generationRequests.set(key, metadata);
  writeGenerationRequest(key, metadata);
}

export function forgetGenerationRunMetadata(
  projectId: string,
  runIds: string[],
) {
  const storedRequests = readGenerationRequests();
  for (const runId of runIds) {
    const key = runKey(projectId, runId);
    generationRequests.delete(key);
    delete storedRequests[key];
  }
  writeGenerationRequests(storedRequests);
}

function coreProjectId(projectId: string) {
  const value = Number(projectId);
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new Error("Asset generation requires a persisted Core API project.");
  }
  return value;
}

function coreAssetId(assetId: string) {
  const value = Number(assetId);
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new Error("Asset generation requires a persisted Core API asset.");
  }
  return value;
}

function readGenerationRequests() {
  if (typeof localStorage === "undefined") return {};
  try {
    const value = JSON.parse(
      localStorage.getItem(generationRequestsStorageKey) ?? "{}",
    );
    return value && typeof value === "object"
      ? (value as Record<string, GenerationRequestMetadata>)
      : {};
  } catch {
    return {};
  }
}

function writeGenerationRequest(
  key: string,
  metadata: GenerationRequestMetadata,
) {
  if (typeof localStorage === "undefined") return;
  try {
    const requests = readGenerationRequests();
    requests[key] = metadata;
    writeGenerationRequests(requests);
  } catch {
    // Metadata is an enhancement; generation itself must still succeed.
  }
}

function writeGenerationRequests(
  requests: Record<string, GenerationRequestMetadata>,
) {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(
      generationRequestsStorageKey,
      JSON.stringify(requests),
    );
  } catch {
    // Ignore storage failures; the in-memory map is still pruned.
  }
}
