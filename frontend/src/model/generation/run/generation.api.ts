import { readFileAsDataUrl } from "@/lib/read-file-as-data-url";

import { assetCanvasSizeDimensionsSchema } from "../../asset/library/asset-canvas-size";
import { coreGenerationApi } from "./core-generation.api";
import type {
  CreateGenerationRequest,
  GenerationRunListItemResponse,
} from "./generation.contract";
import type { CreationRequest, GenerationInput, GenerationRun } from "./types";

type GenerationRequestMetadata = Pick<
  CreationRequest,
  "kind" | "name" | "prompt" | "canvasSize" | "perspective"
>;

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
  listRuns: (projectId: string) => Promise<GenerationRun[]>;
  enqueue: (input: GenerationInput) => Promise<GenerationRun>;
};

export const generationApi: GenerationApi = {
  listRuns: async (projectId) => {
    const response = await coreGenerationApi.list(coreProjectId(projectId), {
      status: "active",
    });
    return response.items.flatMap((item) => {
      const run = toGenerationRun(
        item,
        generationRequests.get(runKey(projectId, item.id)),
      );
      return run ? [run] : [];
    });
  },
  enqueue: async (input) => {
    const request = await toCreateGenerationRequest(input.request);
    const response = await coreGenerationApi.create(
      coreProjectId(input.projectId),
      request,
    );
    const run: GenerationRun = {
      ...toGenerationRequestMetadata(input.request),
      id: String(response.generationRunId),
      projectId: input.projectId,
      status: "pending",
    };
    generationRequests.set(
      runKey(input.projectId, run.id),
      toGenerationRequestMetadata(input.request),
    );
    return run;
  },
};

const generationRequests = new Map<string, GenerationRequestMetadata>();

export function pruneGenerationRequests(
  projectId: string,
  visibleRunIds: string[],
) {
  const visibleKeys = new Set(
    visibleRunIds.map((runId) => runKey(projectId, runId)),
  );
  const projectPrefix = `${projectId}:`;
  for (const key of generationRequests.keys()) {
    if (key.startsWith(projectPrefix) && !visibleKeys.has(key)) {
      generationRequests.delete(key);
    }
  }
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
  const kind = generationKindToAssetKind(item.kind);
  if (!kind || !isVisibleGenerationStatus(item.status)) return undefined;

  return {
    kind,
    name: request?.name ?? `New ${kind}`,
    prompt: request?.prompt ?? "",
    canvasSize: request?.canvasSize ?? "32 × 32 px",
    perspective: request?.perspective,
    id: String(item.id),
    projectId: String(item.projectId),
    status: item.status,
  };
}

function generationKindToAssetKind(
  kind: GenerationRunListItemResponse["kind"],
) {
  if (kind === "generate_character_prototype") return "character" as const;
  if (kind === "generate_object_prototype") return "object" as const;
  if (kind === "generate_tileset") return "tileset" as const;
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

function coreProjectId(projectId: string) {
  const value = Number(projectId);
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new Error("Asset generation requires a persisted Core API project.");
  }
  return value;
}
