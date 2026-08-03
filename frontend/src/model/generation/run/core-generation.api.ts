import type {
  CancelGenerationResponse,
  CreateGenerationRequest,
  CreateGenerationResponse,
  GenerationRunResponse,
  ListGenerationRunsQuery,
  ListGenerationRunsResponse,
} from "./generation.contract";
import { getEnvelope, postEnvelope } from "@/model/fetchers";

export const coreGenerationApi = {
  create: (projectID: number, request: CreateGenerationRequest) =>
    postEnvelope<CreateGenerationResponse>(
      `/projects/${projectID}/generation-runs`,
      request,
    ),
  list: (projectID: number, query?: ListGenerationRunsQuery) =>
    getEnvelope<ListGenerationRunsResponse>(
      `/projects/${projectID}/generation-runs`,
      query,
    ),
  detail: (runID: number) =>
    getEnvelope<GenerationRunResponse>(`/generation-runs/${runID}`),
  cancel: (runID: number) =>
    postEnvelope<CancelGenerationResponse>(`/generation-runs/${runID}/cancel`),
};
