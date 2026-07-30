import { enqueueMockGeneration, listMockGenerationRuns } from "./mock";
import type { GenerationInput, GenerationRun } from "@/model";
import { getJson, postJson } from "@/api/fetchers";

export type GenerationTaskType =
  | "generate_character_prototype"
  | "generate_character_animation"
  | "regenerate_character_prototype"
  | "regenerate_character_animation"
  | "regenerate_character_frames"
  | "generate_object_prototype"
  | "generate_object_animation"
  | "regenerate_object_prototype"
  | "regenerate_object_animation"
  | "regenerate_object_frames"
  | "generate_tileset"
  | "regenerate_item"
  | "regenerate_tiles";
export type GenerationTaskStatus = 0 | 1 | 2 | 3 | 4;
export type CreateGenerationRequest = {
  assetId?: number;
  kind: GenerationTaskType;
  prompt: string;
  referenceMediaIds?: string[];
  targetAssetResourceIds?: number[];
  parameters?: Record<string, unknown>;
};
export type CreateGenerationResponse = { generationRunId: number };
export type ListGenerationRunsQuery = {
  assetId?: number;
  status?: "active";
  limit?: number;
  cursor?: string;
};
export type GenerationRunListItemResponse = {
  id: number;
  projectId: number;
  assetId?: number;
  kind: GenerationTaskType;
  status: GenerationTaskStatus;
};
export type ListGenerationRunsResponse = {
  items: GenerationRunListItemResponse[];
  nextCursor?: string;
};
/** Matches core-api/internal/dto.GetGenerationResponse. */
export type GenerationRunResponse = GenerationRunListItemResponse & {
  result?: unknown;
  error?: string;
};
export type CancelGenerationResponse = { cancelled: boolean };

export type GenerationApi = {
  listRuns: (projectId: string) => Promise<GenerationRun[]>;
  enqueue: (input: GenerationInput) => Promise<GenerationRun>;
};

export const generationApi: GenerationApi = {
  listRuns: (projectId: string) => listMockGenerationRuns(projectId),
  enqueue: enqueueMockGeneration,
};

/** HTTP client for task-backed generation routes. */
export const coreGenerationApi = {
  create: (projectID: number, request: CreateGenerationRequest) =>
    postJson<CreateGenerationResponse>(
      `/projects/${projectID}/generation-runs`,
      request,
    ),
  list: (projectID: number, query?: ListGenerationRunsQuery) =>
    getJson<ListGenerationRunsResponse>(
      `/projects/${projectID}/generation-runs`,
      query,
    ),
  detail: (runID: number) =>
    getJson<GenerationRunResponse>(`/generation-runs/${runID}`),
  cancel: (runID: number) =>
    postJson<CancelGenerationResponse>(`/generation-runs/${runID}/cancel`),
};
