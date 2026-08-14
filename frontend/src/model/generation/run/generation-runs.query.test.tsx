// @vitest-environment happy-dom

import type { PropsWithChildren } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useAssetLibraryQuery } from "../../asset/library/asset-library.query";
import { recordKeys } from "../../asset/record/record.keys";
import type { GenerationRun } from "./types";

const mocks = vi.hoisted(() => ({
  listGroups: vi.fn(),
  listRuns: vi.fn(),
  runDetail: vi.fn(),
  forgetGenerationRunMetadata: vi.fn(),
}));

vi.mock("@/model/auth", () => ({ readAuthenticatedUserId: () => 7 }));
vi.mock("../../asset/library/asset.api", () => ({
  assetApi: { listGroups: mocks.listGroups },
}));
vi.mock("./generation.api", () => ({
  generationApi: { listRuns: mocks.listRuns },
  forgetGenerationRunMetadata: mocks.forgetGenerationRunMetadata,
}));
vi.mock("./core-generation.api", () => ({
  coreGenerationApi: { detail: mocks.runDetail },
}));

import { useGenerationRunsQuery } from "./generation-runs.query";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.listGroups.mockResolvedValue([]);
  mocks.listRuns.mockResolvedValue([]);
  mocks.runDetail.mockResolvedValue({ id: 1, status: "completed" });
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
    expect(mocks.runDetail).toHaveBeenCalledWith(1);
  });

  it("queries and refreshes an asset-scoped animation run", async () => {
    mocks.listRuns
      .mockResolvedValueOnce([generationRun("processing")])
      .mockResolvedValueOnce([]);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    });
    const recordKey = recordKeys.detail(7, "42", "9");
    const loadRecord = vi.fn().mockResolvedValue({ record: "refreshed" });
    queryClient.setQueryDefaults(recordKey, { queryFn: loadRecord });
    queryClient.setQueryData(recordKey, { record: "initial" });
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useGenerationRunsQuery("42", "9"), {
      wrapper,
    });

    await waitFor(() => expect(result.current.data).toHaveLength(1));
    expect(mocks.listRuns).toHaveBeenCalledWith("42", "9");
    await act(() => result.current.refetch());

    await waitFor(() => expect(loadRecord).toHaveBeenCalledOnce());
    expect(mocks.runDetail).toHaveBeenCalledWith(1);
    expect(mocks.forgetGenerationRunMetadata).toHaveBeenCalledWith("42", ["1"]);
  });

  it("keeps a failed animation run with its backend error", async () => {
    mocks.listRuns.mockResolvedValue([generationRun("failed")]);
    mocks.runDetail.mockResolvedValue({
      id: 1,
      projectId: 42,
      kind: "generate_animation",
      status: "failed",
      error: "Video provider rejected the request",
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    });
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useGenerationRunsQuery("42", "9"), {
      wrapper,
    });

    await waitFor(() => expect(result.current.data).toHaveLength(1));

    await waitFor(() =>
      expect(result.current.data?.[0]).toMatchObject({
        status: "failed",
        error: "Video provider rejected the request",
      }),
    );
  });

  it("does not retry a completed run when the asset record reload fails", async () => {
    mocks.listRuns
      .mockResolvedValueOnce([generationRun("processing")])
      .mockResolvedValueOnce([]);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    });
    const recordKey = recordKeys.detail(7, "42", "9");
    const loadRecord = vi
      .fn()
      .mockRejectedValue(new Error("record unavailable"));
    queryClient.setQueryDefaults(recordKey, { queryFn: loadRecord });
    queryClient.setQueryData(recordKey, { record: "initial" });
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useGenerationRunsQuery("42", "9"), {
      wrapper,
    });

    await waitFor(() => expect(result.current.data).toHaveLength(1));
    await act(() => result.current.refetch());

    await waitFor(() => expect(loadRecord).toHaveBeenCalledOnce());
    await waitFor(() => expect(result.current.data).toEqual([]));
    await act(() => result.current.refetch());

    expect(loadRecord).toHaveBeenCalledOnce();
    expect(mocks.runDetail).toHaveBeenCalledOnce();
    expect(mocks.forgetGenerationRunMetadata).toHaveBeenCalledWith("42", ["1"]);
  });

  it("bounds reconciliation retries when run details stay unavailable", async () => {
    mocks.listRuns
      .mockResolvedValueOnce([generationRun("processing")])
      .mockResolvedValue([]);
    mocks.runDetail.mockRejectedValue(new Error("run unavailable"));
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    });
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useGenerationRunsQuery("42", "9"), {
      wrapper,
    });

    await waitFor(() => expect(result.current.data).toHaveLength(1));
    for (let attempt = 1; attempt <= 3; attempt += 1) {
      await act(() => result.current.refetch());
      await waitFor(() =>
        expect(mocks.runDetail).toHaveBeenCalledTimes(attempt),
      );
    }
    await waitFor(() => expect(result.current.data).toEqual([]));

    await act(() => result.current.refetch());
    expect(mocks.runDetail).toHaveBeenCalledTimes(3);
  });
});

function generationRun(status: GenerationRun["status"]): GenerationRun {
  return {
    id: "1",
    projectId: "42",
    status,
    kind: "character",
    name: "Hero",
    prompt: "Hero",
    canvasSize: "64 x 64 px",
  };
}
