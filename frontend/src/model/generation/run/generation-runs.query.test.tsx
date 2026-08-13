// @vitest-environment happy-dom

import type { PropsWithChildren } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useAssetLibraryQuery } from "../../asset/library/asset-library.query";
import type { GenerationRun } from "./types";

const mocks = vi.hoisted(() => ({
  listGroups: vi.fn(),
  listRuns: vi.fn(),
  pruneGenerationRequests: vi.fn(),
}));

vi.mock("@/model/auth", () => ({ readAuthenticatedUserId: () => 7 }));
vi.mock("../../asset/library/asset.api", () => ({
  assetApi: { listGroups: mocks.listGroups },
}));
vi.mock("./generation.api", () => ({
  generationApi: { listRuns: mocks.listRuns },
  pruneGenerationRequests: mocks.pruneGenerationRequests,
}));

import { useGenerationRunsQuery } from "./generation-runs.query";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.listGroups.mockResolvedValue([]);
  mocks.listRuns.mockResolvedValue([]);
});

describe("useGenerationRunsQuery", () => {
  it("reloads the mounted asset library when a generation run settles", async () => {
    mocks.listRuns
      .mockResolvedValueOnce([generationRun("processing")])
      .mockResolvedValueOnce([]);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    });
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(
      () => ({
        assets: useAssetLibraryQuery("42"),
        generations: useGenerationRunsQuery("42"),
      }),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.generations.data).toHaveLength(1);
      expect(mocks.listGroups).toHaveBeenCalledOnce();
    });

    await act(() => result.current.generations.refetch());

    await waitFor(() => expect(mocks.listGroups).toHaveBeenCalledTimes(2));
  });
});

function generationRun(status: GenerationRun["status"]): GenerationRun {
  return {
    id: "run-1",
    projectId: "42",
    status,
    kind: "character",
    name: "Hero",
    prompt: "Hero",
    canvasSize: "64 x 64 px",
  };
}
