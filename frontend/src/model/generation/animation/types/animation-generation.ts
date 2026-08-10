import type {
  CharacterAnimationClip,
  CharacterSpriteSheet,
} from "@/model/asset";

export type SpriteAssetKind = "character" | "object";

export type GenerateAnimationRequest = {
  label: string;
  prompt: string;
};

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
