import type {
  AssetCanvasPosition,
  AssetRevision,
  CharacterAnimation,
  CharacterSpriteSheet,
  GenerateAnimationRequest,
  SpriteAssetKind,
} from "@/model";

import type { EditorGenerationTask } from "../Header/editor-header";
import type { InspectorSubmitRequest } from "../Inspector/inspector.types";

export type SpriteEditorModeProps = {
  header: {
    assetKind: SpriteAssetKind;
    assetName: string;
    version: string;
    projectName: string;
    status: string;
    canUndo: boolean;
    canRedo: boolean;
    isDirty: boolean;
    isSaving: boolean;
    generationTasks: EditorGenerationTask[];
    onBack: () => void;
    onUndo: () => void;
    onRedo: () => void;
    onSave: () => void;
  };
  sprite: {
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
