// @vitest-environment happy-dom

import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  applicationMutation: { isPending: false, mutateAsync: vi.fn() },
  enqueueMutation: { mutateAsync: vi.fn() },
}));

vi.mock("@/model", () => ({
  isGenerationRunActive: (run: { status: string }) =>
    run.status === "pending" || run.status === "processing",
  useEnqueueAssetEditGenerationMutation: () => mocks.enqueueMutation,
  useGenerationCandidateQuery: () => ({ data: undefined }),
  useGenerationRunsQuery: () => ({ data: [] }),
  useResolveGenerationApplicationMutation: () => mocks.applicationMutation,
}));

import { useGenerationEditFlow } from "./use-generation-edit-flow";

afterEach(cleanup);

beforeEach(() => {
  vi.clearAllMocks();
  mocks.enqueueMutation.mutateAsync.mockResolvedValue({ id: "31" });
});

describe("useGenerationEditFlow", () => {
  it("drops a second submission while the first one is active", async () => {
    let resolveCreate!: (value: { id: string }) => void;
    mocks.enqueueMutation.mutateAsync.mockImplementationOnce(
      () =>
        new Promise<{ id: string }>((resolve) => {
          resolveCreate = resolve;
        }),
    );

    const { result } = renderHook(() =>
      useGenerationEditFlow({
        projectId: "7",
        assetId: "8",
        assetKind: "tileset",
        assetName: "Forest",
      }),
    );
    const request = { kind: "add_tileset_item" } as never;
    let firstSubmission!: Promise<boolean>;

    await act(async () => {
      firstSubmission = result.current.submit({
        request,
        prompt: "Add an oak tree",
      });
      await Promise.resolve();
    });

    expect(result.current.isSubmitting).toBe(true);
    await expect(
      act(() =>
        result.current.submit({
          request,
          prompt: "Add a pine tree",
        }),
      ),
    ).resolves.toBe(false);

    resolveCreate({ id: "31" });
    await act(async () => {
      await firstSubmission;
    });

    expect(mocks.enqueueMutation.mutateAsync).toHaveBeenCalledTimes(1);
  });
});
