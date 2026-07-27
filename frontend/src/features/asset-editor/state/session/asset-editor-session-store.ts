import { createStore } from "zustand";
import { temporal } from "zundo";

import type {
  EditorCanvasPosition,
  EditorCharacterAnimationClip,
  EditorDocument,
} from "../../domain";

import type {
  AssetEditorCommand,
  AssetEditorSaveState,
  AssetEditorSessionSnapshot,
} from "./AssetEditorSession.interface";

type AssetEditorSessionState = {
  document: EditorDocument;
  savedDocument: EditorDocument;
  setPrompt: (prompt: string) => void;
  setCharacterNodePosition: (
    nodeId: string,
    position: EditorCanvasPosition,
  ) => void;
  addCharacterAnimation: (label: string) => void;
};

export function createAssetEditorSessionStore(initialDocument: EditorDocument) {
  const document = structuredClone(initialDocument);

  return createStore<AssetEditorSessionState>()(
    temporal(
      (set) => ({
        document,
        savedDocument: structuredClone(initialDocument),
        setPrompt: (prompt) =>
          set((state) => ({ document: { ...state.document, prompt } })),
        setCharacterNodePosition: (nodeId, position) =>
          set((state) => {
            if (state.document.mode !== "character") {
              throw new Error(
                "Character node positions require a character record document.",
              );
            }

            return {
              document: {
                ...state.document,
                character: {
                  ...state.document.character,
                  nodePositions: {
                    ...state.document.character.nodePositions,
                    [nodeId]: position,
                  },
                },
              },
            };
          }),
        addCharacterAnimation: (label) =>
          set((state) => {
            if (state.document.mode !== "character") {
              throw new Error(
                "Character animations require a character record document.",
              );
            }

            const normalizedLabel = label.trim();
            if (!normalizedLabel) return state;

            const animations = state.document.character.animations ?? [];
            const animation: EditorCharacterAnimationClip = {
              id: createCharacterAnimationId(normalizedLabel, animations),
              label: normalizedLabel,
              frameCount: 1,
            };
            return {
              document: {
                ...state.document,
                character: {
                  ...state.document.character,
                  animations: [...animations, animation],
                },
              },
            };
          }),
      }),
      {
        limit: 100,
        partialize: (state) => ({
          document: state.document,
        }),
      },
    ),
  );
}

export type AssetEditorSessionStore = ReturnType<
  typeof createAssetEditorSessionStore
>;

export function resetAssetEditorSessionStore(
  store: AssetEditorSessionStore,
  document: EditorDocument,
) {
  store.setState({
    document: structuredClone(document),
    savedDocument: structuredClone(document),
  });
  store.temporal.getState().clear();
}

export function markAssetEditorSessionSaved(
  store: AssetEditorSessionStore,
  document: EditorDocument,
) {
  store.setState({ savedDocument: structuredClone(document) });
}

export function dispatchAssetEditorCommand(
  store: AssetEditorSessionStore,
  command: AssetEditorCommand,
) {
  switch (command.type) {
    case "prompt.set":
      store.getState().setPrompt(command.value);
      return;
    case "character.node-position.set":
      store
        .getState()
        .setCharacterNodePosition(command.nodeId, command.position);
      return;
    case "character.animation.add":
      store.getState().addCharacterAnimation(command.label);
      return;
    case "history.undo":
      store.temporal.getState().undo();
      return;
    case "history.redo":
      store.temporal.getState().redo();
  }
}

export function getAssetEditorSessionSnapshot(
  store: AssetEditorSessionStore,
  saveState: AssetEditorSaveState,
): AssetEditorSessionSnapshot {
  const state = store.getState();
  const temporalState = store.temporal.getState();

  return {
    document: state.document,
    dirty:
      JSON.stringify(state.document) !== JSON.stringify(state.savedDocument),
    canUndo: temporalState.pastStates.length > 0,
    canRedo: temporalState.futureStates.length > 0,
    saveState,
  };
}

function createCharacterAnimationId(
  label: string,
  animations: Array<{ id: string }>,
) {
  const base =
    label
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "") || "animation";
  const ids = new Set(animations.map((animation) => animation.id));
  let id = base;
  let suffix = 2;

  while (ids.has(id)) {
    id = `${base}-${suffix}`;
    suffix += 1;
  }

  return id;
}
