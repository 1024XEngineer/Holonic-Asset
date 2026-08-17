import type {
  CancelGenerationResponse,
  CreateGenerationRequest,
  CreateGenerationResponse,
  GenerationRunResponse,
  ListGenerationRunsQuery,
  ListGenerationRunsResponse,
  ResolveGenerationApplicationResponse,
} from "./generation.contract";
import { coreApiClient, unwrapApiResponse } from "@/model/fetchers";

export const coreGenerationApi = {
  create: async (projectID: number, request: CreateGenerationRequest) =>
    unwrapApiResponse<CreateGenerationResponse>(
      await coreApiClient.POST("/projects/{project_id}/generation-runs", {
        params: { path: { project_id: projectID } },
        body: request,
      }),
    ),
  list: async (projectID: number, query?: ListGenerationRunsQuery) =>
    unwrapApiResponse<ListGenerationRunsResponse>(
      await coreApiClient.GET("/projects/{project_id}/generation-runs", {
        params: { path: { project_id: projectID }, query },
      }),
    ),
  detail: async (runID: number) =>
    unwrapApiResponse<GenerationRunResponse>(
      await coreApiClient.GET("/generation-runs/{run_id}", {
        params: { path: { run_id: runID } },
      }),
    ),
  cancel: async (runID: number) =>
    unwrapApiResponse<CancelGenerationResponse>(
      await coreApiClient.POST("/generation-runs/{run_id}/cancel", {
        params: { path: { run_id: runID } },
      }),
    ),
  resolveApplication: async (runID: number, applied: boolean) =>
    unwrapApiResponse<ResolveGenerationApplicationResponse>(
      await coreApiClient.POST("/generation-runs/{run_id}/application", {
        params: { path: { run_id: runID } },
        body: { applied },
      }),
    ),
};
