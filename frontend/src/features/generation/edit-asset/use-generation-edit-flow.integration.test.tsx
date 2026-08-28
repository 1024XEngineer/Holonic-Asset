// @vitest-environment happy-dom

import type { PropsWithChildren } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { GenerationRun } from "@/model";

const mocks = vi.hoisted(() => ({
  coreCreate: vi.fn(),
  listRuns: vi.fn(),
  rememberGenerationRunMetadata: vi.fn(),
}));

vi.mock("@/model/auth", () => ({ readAuthenticatedUserId: () => 7 }));
vi.mock("@/model/generation/run/core-generation.api", () => ({
  coreGenerationApi: {
    create: mocks.coreCreate,
    detail: vi.fn(),
    resolveApplication: vi.fn(),
  },
}));
vi.mock("@/model/generation/run/generation.api", () => ({
  coreGenerationApi: { create: mocks.coreCreate },
  generationApi: { listRuns: mocks.listRuns },
  forgetGenerationRunMetadata: vi.fn(),
  rememberGenerationRunMetadata: mocks.rememberGenerationRunMetadata,
}));

import { useGenerationEditFlow } from "./use-generation-edit-flow";

afterEach(cleanup);

beforeEach(() => {
  vi.clearAllMocks();
  mocks.coreCreate.mockResolvedValue({ generationRunId: 31 });
  mocks.listRuns.mockResolvedValue([]);
});

describe("useGenerationEditFlow integration", () => {
  it("shows a newly submitted edit in the asset generation queue", async () => {
    const { result } = renderHook(
      () =>
        useGenerationEditFlow({
          projectId: "7",
          assetId: "8",
          assetKind: "tileset",
          assetName: "Forest",
        }),
      { wrapper: queryWrapper() },
    );
    await waitFor(() => expect(mocks.listRuns).toHaveBeenCalledOnce());

    await act(() =>
      result.current.submit({
        request: { kind: "edit_tileset_item" } as never,
        prompt: "Add moss",
      }),
    );

    await waitFor(() =>
      expect(result.current.runs).toContainEqual(
        expect.objectContaining({
          id: "31",
          projectId: "7",
          assetId: "8",
          status: "pending",
        }),
      ),
    );
  });

  it("rejects another edit while the asset already has an active run", async () => {
    mocks.listRuns.mockResolvedValue([generationRun("processing")]);
    const { result } = renderHook(
      () =>
        useGenerationEditFlow({
          projectId: "7",
          assetId: "8",
          assetKind: "tileset",
          assetName: "Forest",
        }),
      { wrapper: queryWrapper() },
    );
    await waitFor(() => expect(result.current.runs).toHaveLength(1));

    let submitted = true;
    await act(async () => {
      submitted = await result.current.submit({
        request: { kind: "edit_tileset_item" } as never,
        prompt: "Add moss",
      });
    });

    expect(submitted).toBe(false);
    expect(mocks.coreCreate).not.toHaveBeenCalled();
  });
});

function queryWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false, staleTime: Infinity },
    },
  });
  return ({ children }: PropsWithChildren) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

function generationRun(status: GenerationRun["status"]): GenerationRun {
  return {
    id: "31",
    projectId: "7",
    assetId: "8",
    kind: "tileset",
    name: "Edit Forest",
    prompt: "Add moss",
    canvasSize: "64 x 64 px",
    status,
  };
}
