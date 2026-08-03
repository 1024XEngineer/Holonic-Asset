import { enqueueMockGeneration, listMockGenerationRuns } from "./mock";
import type { GenerationInput } from "./mock/generation-input";
import type { GenerationRun } from "@/features/generation/types";

export type { GenerationInput } from "./mock/generation-input";

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
  listRuns: (projectId: string) => listMockGenerationRuns(projectId),
  enqueue: enqueueMockGeneration,
};
