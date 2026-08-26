// @vitest-environment happy-dom

import type { PropsWithChildren } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { GenerationRun } from "../run/types";
import { generationKeys } from "../run/keys";

const mocks = vi.hoisted(() => ({ generate: vi.fn() }));

vi.mock("@/model/auth", () => ({ readAuthenticatedUserId: () => 7 }));
vi.mock("./animation-generation.api", () => ({
  animationGenerationApi: { generate: mocks.generate },
}));

import { useGenerateAnimationMutation } from "./animation-generation.mutation";

beforeEach(() => vi.clearAllMocks());

describe("useGenerateAnimationMutation", () => {
  it("keeps the created run in the asset queue without a loading gap", async () => {
    const run = animationRun();
    mocks.generate.mockResolvedValue(run);
    const queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    });
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useGenerateAnimationMutation(), {
      wrapper,
    });

    await act(() => result.current.mutateAsync(animationInput()));

    expect(
      queryClient.getQueryData<GenerationRun[]>(
        generationKeys.runs(7, "11", "9"),
      ),
    ).toEqual([run]);
  });
});

function animationRun(): GenerationRun {
  return {
    id: "91",
    projectId: "11",
    assetId: "9",
    kind: "character",
    name: "Walk",
    prompt: "A relaxed walk",
    canvasSize: "32 × 32 px",
    status: "pending",
  };
}

function animationInput() {
  return {
    projectId: "11",
    assetId: "9",
    assetKind: "character" as const,
    animationName: "Walk",
    direction: "left" as const,
    creativeBrief: "A relaxed walk",
    frameCount: 8,
    frameWidth: 48,
    frameHeight: 48,
    fps: 12,
    duration: 5,
  };
}
