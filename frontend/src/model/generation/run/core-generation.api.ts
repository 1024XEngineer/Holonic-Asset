import type {
  CancelGenerationResponse,
  CreateGenerationRequest,
  CreateGenerationResponse,
  DeleteGenerationResponse,
  GenerationRunResponse,
  ListGenerationRunsQuery,
  ListGenerationRunsResponse,
  RetryGenerationResponse,
} from "./generation.contract";
import {
  coreApiClient,
  ensureApiResponseSuccess,
  unwrapApiResponse,
} from "@/model/fetchers";

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
        cache: "no-store",
      }),
    ),
  detail: async <Content = unknown>(runID: number) =>
    unwrapApiResponse<GenerationRunResponse<Content>>(
      await coreApiClient.GET("/generation-runs/{run_id}", {
        params: { path: { run_id: runID } },
        cache: "no-store",
      }),
    ),
  cancel: async (runID: number) =>
    unwrapApiResponse<CancelGenerationResponse>(
      await coreApiClient.POST("/generation-runs/{run_id}/cancel", {
        params: { path: { run_id: runID } },
      }),
    ),
  retry: async (runID: number) =>
    unwrapApiResponse<RetryGenerationResponse>(
      await coreApiClient.POST("/generation-runs/{run_id}/retry", {
        params: { path: { run_id: runID } },
      }),
    ),
  delete: async (runID: number) =>
    unwrapApiResponse<DeleteGenerationResponse>(
      await coreApiClient.DELETE("/generation-runs/{run_id}", {
        params: { path: { run_id: runID } },
      }),
    ),
  resolveApplication: async (runID: number, applied: boolean) =>
    ensureApiResponseSuccess(
      await coreApiClient.POST("/generation-runs/{run_id}/application", {
        params: { path: { run_id: runID } },
        body: { applied },
      }),
    ),
};
