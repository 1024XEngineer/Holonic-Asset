import type {
  CharacterAnimationClip,
  CharacterSpriteSheet,
} from "@/model/asset";
import { z } from "zod";

export const generateAnimationRequestSchema = z.object({
  label: z.string().trim().min(1, "Animation name is required."),
  prompt: z.string().trim().min(1, "Generation prompt is required."),
});

export type SpriteAssetKind = "character" | "object";
export type GenerateAnimationRequest = z.infer<
  typeof generateAnimationRequestSchema
>;
export type GenerateAnimationInput = GenerateAnimationRequest & {
  projectId: string;
  assetId: string;
  assetKind: SpriteAssetKind;
  prototype: CharacterSpriteSheet;
};

export type GeneratedCharacterAnimation = Omit<CharacterAnimationClip, "id">;

export type GenerateAnimationResult = {
  generationId: string;
  animation: GeneratedCharacterAnimation;
};
