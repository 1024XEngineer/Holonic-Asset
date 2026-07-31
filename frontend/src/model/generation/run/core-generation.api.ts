import type {
  CancelGenerationResponse,
  CreateGenerationRequest,
  CreateGenerationResponse,
  GenerationRunResponse,
  ListGenerationRunsQuery,
  ListGenerationRunsResponse,
} from "./generation.contract";
import { getJson, postJson } from "@/model/fetchers";

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
