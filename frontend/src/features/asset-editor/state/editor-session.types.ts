import type {
  AssetCanvasPosition,
  AssetRecord,
  GeneratedCharacterAnimation,
} from "@/model";

export type EditorTarget = {
  projectId: string;
  assetId: string;
};

export type EditorCommand =
  | { type: "prompt.set"; value: string }
  | {
      type: "character.node-position.set";
      nodeId: string;
      position: AssetCanvasPosition;
    }
  | {
      type: "character.animation.generated";
      animation: GeneratedCharacterAnimation;
    }
  | {
      type: "character.animation.rename";
      animationId: string;
      label: string;
    }
  | { type: "character.animation.delete"; animationId: string }
  | { type: "history.undo" }
  | { type: "history.redo" };

export type EditorSaveState =
  | { phase: "idle" }
  | { phase: "saving" }
  | { phase: "failed"; message: string };

export type EditorSessionSnapshot = {
  record: AssetRecord;
  dirty: boolean;
  canUndo: boolean;
  canRedo: boolean;
  saveState: EditorSaveState;
};

export type EditorSaveResult =
  | { status: "saved" }
  | { status: "failed" }
  | { status: "superseded" };

export type EditorSession = {
  snapshot: EditorSessionSnapshot;
  dispatch: (command: EditorCommand) => void;
  save: () => Promise<EditorSaveResult>;
};

export type UseEditorSessionInput = {
  target: EditorTarget;
  initialRecord: AssetRecord;
};
