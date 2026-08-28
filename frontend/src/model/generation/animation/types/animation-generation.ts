import { assetDirectionSchema, type AssetDirection } from "@/model/asset";
import { z } from "zod";

export const generateAnimationRequestSchema = z.object({
  animationName: z.string().trim().min(1, "Animation name is required."),
  direction: assetDirectionSchema,
  creativeBrief: z.string().trim().min(1, "Creative brief is required."),
  frameCount: z.number().int().min(1).max(32),
  frameWidth: z.number().int().min(32).max(1024),
  frameHeight: z.number().int().min(32).max(1024),
  fps: z.number().int().min(1).max(60),
  duration: z.number().int().min(4).max(15),
});

export const deriveAnimationRequestSchema = z.object({
  sourceAnimationId: z.string().trim().min(1),
  sourceAnimationName: z.string().trim().min(1),
  targetDirections: z.array(assetDirectionSchema).min(1),
});

export type SpriteAssetKind = "character" | "object";
export type GenerateAnimationRequest = z.infer<
  typeof generateAnimationRequestSchema
>;
export type DeriveAnimationRequest = z.infer<
  typeof deriveAnimationRequestSchema
>;
export type GenerateAnimationInput = GenerateAnimationRequest & {
  projectId: string;
  assetId: string;
  assetKind: SpriteAssetKind;
};
export type DeriveAnimationInput = DeriveAnimationRequest & {
  projectId: string;
  assetId: string;
  assetKind: SpriteAssetKind;
};

export type { AssetDirection };
