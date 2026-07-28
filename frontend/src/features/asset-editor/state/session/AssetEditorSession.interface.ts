import type { EditorCanvasPosition, EditorRecord } from "../../domain";

export type AssetEditorTarget = {
  projectId: string;
  assetId: string;
};

export type AssetEditorCommand =
  | {
      type: "prompt.set";
      value: string;
    }
  | {
      type: "character.node-position.set";
      nodeId: string;
      position: EditorCanvasPosition;
    }
  | {
      type: "character.animation.add";
      label: string;
    }
  | {
      type: "history.undo";
    }
  | {
      type: "history.redo";
    };

export type AssetEditorSaveState =
  | { phase: "idle" }
  | { phase: "saving" }
  | { phase: "failed"; message: string };

export type AssetEditorSessionSnapshot = {
  record: EditorRecord;
  dirty: boolean;
  canUndo: boolean;
  canRedo: boolean;
  saveState: AssetEditorSaveState;
};

export type AssetEditorSaveResult =
  | { status: "saved" }
  | { status: "failed" }
  | { status: "superseded" };

export type AssetEditorSession = {
  snapshot: AssetEditorSessionSnapshot;
  dispatch: (command: AssetEditorCommand) => void;
  save: () => Promise<AssetEditorSaveResult>;
};

export type UseAssetEditorSessionInput = {
  target: AssetEditorTarget;
  initialRecord: EditorRecord;
};
