import { readFileAsDataUrl } from "@/lib/read-file-as-data-url";

import {
  assetCanvasSizeDimensionsSchema,
  defaultAssetCanvasSize,
} from "../../asset/library/asset-canvas-size";
import type { CreatableAssetKind } from "../../asset";
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
    const runs = response.items.flatMap((item) => {
      const run = toGenerationRun(
        item,
        generationRequests.get(runKey(projectId, item.id)) ??
          storedMetadata[runKey(projectId, item.id)],
      );
      return run ? [run] : [];
    });
    return runs;
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
const generationRequestsStorageKey = "holonic-generation-requests";

export function pruneGenerationRequests(
  projectId: string,
  visibleRunIds: string[],
  assetId?: string,
) {
  const visibleKeys = new Set(
    visibleRunIds.map((runId) => runKey(projectId, runId)),
  );
  const projectPrefix = `${projectId}:`;
  const storedRequests = readGenerationRequests();
  const keys = new Set([
    ...generationRequests.keys(),
    ...Object.keys(storedRequests),
  ]);
  for (const key of keys) {
    const metadata = generationRequests.get(key) ?? storedRequests[key];
    if (
      key.startsWith(projectPrefix) &&
      metadata?.assetId === assetId &&
      !visibleKeys.has(key)
    ) {
      generationRequests.delete(key);
      delete storedRequests[key];
    }
  }
  writeGenerationRequests(storedRequests);
}

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
): GenerationRun | undefined {
  const kind = generationKindToAssetKind(item.kind, request?.kind);
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
  return undefined;
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
