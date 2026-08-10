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
  assetKind: SpriteAssetKind;
  assetName: string;
  version: string;
  projectName: string;
  prototype: CharacterSpriteSheet;
  animations: CharacterAnimation[];
  nodePositions: Record<string, AssetCanvasPosition>;
  prompt: string;
  history: AssetRevision[];
  status: string;
  canUndo: boolean;
  canRedo: boolean;
  isDirty: boolean;
  isSaving: boolean;
  isPromptSubmitting: boolean;
  promptSubmitError: string | null;
  isGeneratingAnimation: boolean;
  generationTasks: EditorGenerationTask[];
  onBack: () => void;
  onUndo: () => void;
  onRedo: () => void;
  onSave: () => void;
  onPromptChange: (value: string) => void;
  onPromptSubmit: (request: InspectorSubmitRequest) => void | Promise<void>;
  onPositionChange: (nodeId: string, position: AssetCanvasPosition) => void;
  onAnimationGenerate: (request: GenerateAnimationRequest) => void;
  onAnimationRename: (animationId: string, label: string) => void;
  onAnimationDelete: (animationId: string) => void;
};
