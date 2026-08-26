import { defaultAssetCanvasSize } from "../../asset/library/asset-canvas-size";
import type { GenerationRun } from "../run/types";
import type { GenerateAnimationInput } from "./types";
import { coreGenerationApi } from "../run/core-generation.api";
import { rememberGenerationRunMetadata } from "../run/generation.api";

export type AnimationGenerationApi = {
  generate: (input: GenerateAnimationInput) => Promise<GenerationRun>;
};

export const animationGenerationApi: AnimationGenerationApi = {
  generate: async (input) => {
    const projectId = positiveCoreId(input.projectId, "project");
    const assetId = positiveCoreId(input.assetId, "asset");
    const response = await coreGenerationApi.create(projectId, {
      kind: "generate_animation",
      assetId,
      creative_brief: input.creativeBrief,
      parameters: {
        animation_name: input.animationName,
        direction: input.direction,
        frame_count: input.frameCount,
        frame_width: input.frameWidth,
        frame_height: input.frameHeight,
        fps: input.fps,
        duration: input.duration,
      },
    });

    rememberGenerationRunMetadata(input.projectId, response.generationRunId, {
      kind: input.assetKind,
      name: input.animationName,
      prompt: input.creativeBrief,
      assetId: input.assetId,
    });
    return {
      id: String(response.generationRunId),
      projectId: input.projectId,
      assetId: input.assetId,
      kind: input.assetKind,
      name: input.animationName,
      prompt: input.creativeBrief,
      canvasSize: defaultAssetCanvasSize,
      status: "pending",
    };
  },
};

function positiveCoreId(value: string, resource: "project" | "asset") {
  const id = Number(value);
  if (!Number.isSafeInteger(id) || id <= 0) {
    throw new Error(
      `Animation generation requires a persisted Core API ${resource}.`,
    );
  }
  return id;
}
