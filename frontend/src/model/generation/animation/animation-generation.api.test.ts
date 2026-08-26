import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  rememberMetadata: vi.fn(),
}));

vi.mock("../run/core-generation.api", () => ({
  coreGenerationApi: { create: mocks.create },
}));
vi.mock("../run/generation.api", () => ({
  rememberGenerationRunMetadata: mocks.rememberMetadata,
}));

import { animationGenerationApi } from "./animation-generation.api";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.create.mockResolvedValue({ generationRunId: 91 });
});

describe("animationGenerationApi", () => {
  it("queues the simplified animation payload through the generation API", async () => {
    await expect(
      animationGenerationApi.generate({
        projectId: "11",
        assetId: "7",
        assetKind: "character",
        animationName: "Walk",
        direction: "back_left",
        creativeBrief: "A relaxed looping walk with grounded footfalls",
        frameCount: 8,
        frameWidth: 48,
        frameHeight: 48,
        fps: 12,
        duration: 5,
      }),
    ).resolves.toEqual({
      id: "91",
      projectId: "11",
      assetId: "7",
      kind: "character",
      name: "Walk",
      prompt: "A relaxed looping walk with grounded footfalls",
      canvasSize: "32 × 32 px",
      status: "pending",
    });

    expect(mocks.create).toHaveBeenCalledWith(11, {
      kind: "generate_animation",
      assetId: 7,
      creative_brief: "A relaxed looping walk with grounded footfalls",
      parameters: {
        animation_name: "Walk",
        direction: "back_left",
        frame_count: 8,
        frame_width: 48,
        frame_height: 48,
        fps: 12,
        duration: 5,
      },
    });
    expect(mocks.rememberMetadata).toHaveBeenCalledWith("11", 91, {
      kind: "character",
      name: "Walk",
      prompt: "A relaxed looping walk with grounded footfalls",
      assetId: "7",
    });
  });

  it("includes custom frame dimensions in the generation payload", async () => {
    await animationGenerationApi.generate({
      projectId: "11",
      assetId: "7",
      assetKind: "object",
      animationName: "Open",
      direction: "right",
      creativeBrief: "Open the lid",
      frameCount: 6,
      frameWidth: 96,
      frameHeight: 128,
      fps: 12,
      duration: 4,
    });

    expect(mocks.create).toHaveBeenCalledWith(
      11,
      expect.objectContaining({
        parameters: expect.objectContaining({
          frame_width: 96,
          frame_height: 128,
        }),
      }),
    );
  });

  it.each([
    ["project", "draft-project", "7"],
    ["asset", "11", "draft-asset"],
  ])("rejects a non-persisted %s", async (_resource, projectId, assetId) => {
    await expect(
      animationGenerationApi.generate({
        projectId,
        assetId,
        assetKind: "object",
        animationName: "Open",
        direction: "right",
        creativeBrief: "Open the lid",
        frameCount: 6,
        frameWidth: 48,
        frameHeight: 48,
        fps: 12,
        duration: 4,
      }),
    ).rejects.toThrow("persisted Core API");
    expect(mocks.create).not.toHaveBeenCalled();
  });
});
