import type {
  AssetCanvasPosition,
  AssetRevision,
  CharacterAnimation,
  CharacterSpriteSheet,
  GenerateAnimationRequest,
  Perspective,
} from "@/model";

import type { EditorHeaderProps } from "../Header/editor-header";
import type { InspectorSubmitRequest } from "../Inspector/inspector.types";
import type { GenerationReview } from "../GenerationReview/generation-review-bar";

export type SpriteEditorModeProps = {
  header: EditorHeaderProps;
  generationReview?: GenerationReview;
  sprite: {
    perspective: Perspective;
    prototype: CharacterSpriteSheet;
    animations: CharacterAnimation[];
    nodePositions: Record<string, AssetCanvasPosition>;
    onPositionChange: (nodeId: string, position: AssetCanvasPosition) => void;
  };
  tree: {
    isGeneratingAnimation: boolean;
    onAnimationGenerate: (request: GenerateAnimationRequest) => void;
    onAnimationRename: (animationId: string, label: string) => void;
    onAnimationDelete: (animationId: string) => void;
  };
  inspector: {
    prompt: string;
    history: AssetRevision[];
    isSubmitting: boolean;
    onPromptChange: (value: string) => void;
    onSubmit: (request: InspectorSubmitRequest) => void | Promise<void>;
  };
};
