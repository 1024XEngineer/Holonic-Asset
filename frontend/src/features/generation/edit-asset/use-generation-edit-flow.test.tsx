// @vitest-environment happy-dom

import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  applicationMutation: { isPending: false, mutateAsync: vi.fn() },
  coreCreate: vi.fn(),
  rememberGenerationRunMetadata: vi.fn(),
}));

vi.mock("@/hooks/use-timeout", () => ({
  useTimeout: () => ({ schedule: vi.fn() }),
}));

vi.mock("@/model", () => ({
  coreGenerationApi: { create: mocks.coreCreate },
  rememberGenerationRunMetadata: mocks.rememberGenerationRunMetadata,
  useGenerationCandidateQuery: () => ({ data: undefined }),
  useGenerationRunsQuery: () => ({ data: [] }),
  useResolveGenerationApplicationMutation: () => mocks.applicationMutation,
}));

import { useGenerationEditFlow } from "./use-generation-edit-flow";

afterEach(cleanup);

beforeEach(() => {
  vi.clearAllMocks();
  mocks.coreCreate.mockResolvedValue({ generationRunId: 31 });
});

describe("useGenerationEditFlow", () => {
  it("drops a second submission while the first one is active", async () => {
    let resolveCreate!: (value: { generationRunId: number }) => void;
    mocks.coreCreate.mockImplementationOnce(
      () =>
        new Promise<{ generationRunId: number }>((resolve) => {
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

    resolveCreate({ generationRunId: 31 });
    await act(async () => {
      await firstSubmission;
    });

    expect(mocks.coreCreate).toHaveBeenCalledTimes(1);
  });
});
