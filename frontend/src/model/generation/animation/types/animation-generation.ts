import { assetDirectionSchema } from "@/model/asset";
import { z } from "zod";

export const generateAnimationRequestSchema = z.object({
  animationName: z.string().trim().min(1, "Animation name is required."),
  direction: assetDirectionSchema,
  creativeBrief: z.string().trim().min(1, "Creative brief is required."),
  frameCount: z.number().int().min(1).max(32),
  fps: z.number().int().min(1).max(60),
  duration: z.number().int().min(4).max(15),
});

export type SpriteAssetKind = "character" | "object";
export type GenerateAnimationRequest = z.infer<
  typeof generateAnimationRequestSchema
>;
export type GenerateAnimationInput = GenerateAnimationRequest & {
  projectId: string;
  assetId: string;
  assetKind: SpriteAssetKind;
};
