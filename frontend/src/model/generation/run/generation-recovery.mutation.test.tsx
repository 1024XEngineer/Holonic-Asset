// @vitest-environment happy-dom

import { act, cleanup, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { GenerationRun } from "./types";

const mocks = vi.hoisted(() => ({
  delete: vi.fn(),
  forgetMetadata: vi.fn(),
  retry: vi.fn(),
}));

vi.mock("@/model/auth", () => ({
  readAuthenticatedUserId: () => 7,
}));

vi.mock("./core-generation.api", () => ({
  coreGenerationApi: {
    delete: mocks.delete,
    retry: mocks.retry,
  },
}));

vi.mock("./generation.api", () => ({
  forgetGenerationRunMetadata: mocks.forgetMetadata,
}));

import {
  deleteGenerationRun,
  retryGenerationRun,
  useDeleteGenerationRunMutation,
  useRetryGenerationRunMutation,
} from "./generation-recovery.mutation";
import { generationKeys } from "./keys";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.delete.mockResolvedValue({ deleted: true });
  mocks.retry.mockResolvedValue({ generationRunId: 12 });
});

afterEach(cleanup);

describe("generation recovery", () => {
  it("routes retry and delete through persisted Core API run IDs", async () => {
    const input = { projectId: "42", runId: "12" };

    await expect(retryGenerationRun(input)).resolves.toEqual({
      generationRunId: 12,
    });
    await expect(deleteGenerationRun(input)).resolves.toEqual({
      deleted: true,
    });

    expect(mocks.retry).toHaveBeenCalledWith(12);
    expect(mocks.delete).toHaveBeenCalledWith(12);
  });

  it("rejects recovery for a non-persisted run", async () => {
    const input = { projectId: "42", runId: "draft" };

    await expect(retryGenerationRun(input)).rejects.toThrow(
      "generation run requires a persisted Core API identifier",
    );
    await expect(deleteGenerationRun(input)).rejects.toThrow(
      "generation run requires a persisted Core API identifier",
    );
    expect(mocks.retry).not.toHaveBeenCalled();
    expect(mocks.delete).not.toHaveBeenCalled();
  });

  it("moves a retried run back to pending in every matching queue", async () => {
    const { queryClient, wrapper } = testQueryClient();
    const projectKey = generationKeys.runs(7, "42");
    const assetKey = generationKeys.runs(7, "42", "8");
    queryClient.setQueryData(projectKey, [failedRun()]);
    queryClient.setQueryData(assetKey, [failedRun()]);
    const { result } = renderHook(() => useRetryGenerationRunMutation(), {
      wrapper,
    });

    await act(() =>
      result.current.mutateAsync({ projectId: "42", runId: "12" }),
    );

    expect(queryClient.getQueryData<GenerationRun[]>(projectKey)?.[0]).toEqual(
      expect.objectContaining({ status: "pending", error: undefined }),
    );
    expect(queryClient.getQueryData<GenerationRun[]>(assetKey)?.[0]).toEqual(
      expect.objectContaining({ status: "pending", error: undefined }),
    );
  });

  it("removes a deleted run from every matching queue and its metadata", async () => {
    const { queryClient, wrapper } = testQueryClient();
    const projectKey = generationKeys.runs(7, "42");
    const assetKey = generationKeys.runs(7, "42", "8");
    queryClient.setQueryData(projectKey, [failedRun()]);
    queryClient.setQueryData(assetKey, [failedRun()]);
    const { result } = renderHook(() => useDeleteGenerationRunMutation(), {
      wrapper,
    });

    await act(() =>
      result.current.mutateAsync({ projectId: "42", runId: "12" }),
    );

    expect(queryClient.getQueryData(projectKey)).toEqual([]);
    expect(queryClient.getQueryData(assetKey)).toEqual([]);
    expect(mocks.forgetMetadata).toHaveBeenCalledWith("42", ["12"]);
  });
});

function testQueryClient() {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
  });
  return {
    queryClient,
    wrapper: ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    ),
  };
}

function failedRun(): GenerationRun {
  return {
    id: "12",
    projectId: "42",
    status: "failed",
    error: "provider unavailable",
    kind: "object",
    name: "Lantern",
    prompt: "Brass lantern",
    canvasSize: "64 × 64 px",
  };
}
