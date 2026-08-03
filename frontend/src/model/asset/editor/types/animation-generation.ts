import type {
  EditorCharacterAnimationClip,
  EditorCharacterSpriteSheet,
} from "./editor-record";

export type GenerateAnimationRequest = {
  label: string;
  prompt: string;
};

export type GenerateAnimationInput = GenerateAnimationRequest & {
  projectId: string;
  assetId: string;
  prototype: EditorCharacterSpriteSheet;
};

export type GeneratedEditorCharacterAnimation = Omit<
  EditorCharacterAnimationClip,
  "id"
>;

export type GenerateAnimationResult = {
  generationId: string;
  animation: GeneratedEditorCharacterAnimation;
};
