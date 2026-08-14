import { readFileAsDataUrl } from "@/lib/read-file-as-data-url";

import {
  assetCanvasSizeDimensionsSchema,
  defaultAssetCanvasSize,
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
  GenerationRunListItemResponse,
  GenerationRunResponse,
  GenerationTaskStatus,
  GenerationTaskType,
  ListGenerationRunsQuery,
  ListGenerationRunsResponse,
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
          await resolveAnimationAssetKind(item, request?.kind),
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
  if (request.kind !== "character" && request.kind !== "object") {
    throw new Error(
      "Core API creation currently supports Character and Object assets only.",
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
      reference: await resolveReference(request.reference),
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
    canvasSize: request?.canvasSize ?? defaultAssetCanvasSize,
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
  if (kind === "generate_character_prototype") return "character" as const;
  if (kind === "generate_object_prototype") return "object" as const;
  if (kind === "generate_tileset") return "tileset" as const;
  if (
    kind === "generate_animation" &&
    (requestedKind === "character" || requestedKind === "object")
  ) {
    return requestedKind;
  }
  if (kind === "generate_animation") {
    return resolvedAnimationKind ?? ("character" as const);
  }
  return undefined;
}

async function resolveAnimationAssetKind(
  item: GenerationRunListItemResponse,
  requestedKind: CreatableAssetKind | undefined,
): Promise<"character" | "object" | undefined> {
  if (requestedKind === "character" || requestedKind === "object") {
    return requestedKind;
  }
  if (item.kind !== "generate_animation" || item.assetId === undefined) {
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
  return status === "pending" || status === "processing" || status === "failed";
}

async function resolveReference(reference: unknown) {
  if (reference === undefined) return "";
  if (typeof reference === "string") return reference.trim();
  if (typeof File !== "undefined" && reference instanceof File) {
    return readFileAsDataUrl(reference);
  }
  throw new Error("Reference must be an image file or URL.");
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
