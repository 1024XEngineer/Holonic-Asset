import { createStore } from "zustand";
import { temporal } from "zundo";

import type {
  EditorCanvasPosition,
  EditorRecord,
  GeneratedEditorCharacterAnimation,
} from "@/features/asset-editor/types";

import type {
  AssetEditorCommand,
  AssetEditorSaveState,
  AssetEditorSessionSnapshot,
} from "./AssetEditorSession.interface";

type AssetEditorSessionState = {
  record: EditorRecord;
  savedRecord: EditorRecord;
  setPrompt: (prompt: string) => void;
  setCharacterNodePosition: (
    nodeId: string,
    position: EditorCanvasPosition,
  ) => void;
  addGeneratedCharacterAnimation: (
    animation: GeneratedEditorCharacterAnimation,
  ) => void;
  renameCharacterAnimation: (animationId: string, label: string) => void;
  deleteCharacterAnimation: (animationId: string) => void;
};

export function createAssetEditorSessionStore(initialRecord: EditorRecord) {
  const record = structuredClone(initialRecord);

  return createStore<AssetEditorSessionState>()(
    temporal(
      (set) => ({
        record,
        savedRecord: structuredClone(initialRecord),
        setPrompt: (prompt) =>
          set((state) => ({ record: { ...state.record, prompt } })),
        setCharacterNodePosition: (nodeId, position) =>
          set((state) => {
            if (state.record.mode !== "character") {
              throw new Error(
                "Character node positions require a character record.",
              );
            }

            return {
              record: {
                ...state.record,
                character: {
                  ...state.record.character,
                  nodePositions: {
                    ...state.record.character.nodePositions,
                    [nodeId]: position,
                  },
                },
              },
            };
          }),
        addGeneratedCharacterAnimation: (animation) =>
          set((state) => {
            if (state.record.mode !== "character") {
              throw new Error(
                "Character animations require a character record.",
              );
            }

            const normalizedLabel = animation.label.trim();
            if (!normalizedLabel) return state;

            const animations = state.record.character.animations ?? [];
            return {
              record: {
                ...state.record,
                character: {
                  ...state.record.character,
                  animations: [
                    ...animations,
                    {
                      ...animation,
                      id: createCharacterAnimationId(
                        normalizedLabel,
                        animations,
                      ),
                      label: normalizedLabel,
                    },
                  ],
                },
              },
            };
          }),
        renameCharacterAnimation: (animationId, label) =>
          set((state) => {
            if (state.record.mode !== "character") {
              throw new Error(
                "Character animations require a character record.",
              );
            }

            const normalizedLabel = label.trim();
            if (!normalizedLabel) return state;

            return {
              record: {
                ...state.record,
                character: {
                  ...state.record.character,
                  animations: (state.record.character.animations ?? []).map(
                    (animation) =>
                      animation.id === animationId
                        ? { ...animation, label: normalizedLabel }
                        : animation,
                  ),
                },
              },
            };
          }),
        deleteCharacterAnimation: (animationId) =>
          set((state) => {
            if (state.record.mode !== "character") {
              throw new Error(
                "Character animations require a character record.",
              );
            }

            const animations = state.record.character.animations ?? [];
            const deleted = animations.find(
              (animation) => animation.id === animationId,
            );
            if (!deleted) return state;

            const nodePositions = Object.fromEntries(
              Object.entries(state.record.character.nodePositions).filter(
                ([nodeId]) => nodeId !== deleted.id,
              ),
            );

            return {
              record: {
                ...state.record,
                character: {
                  ...state.record.character,
                  animations: animations.filter(
                    (animation) => animation.id !== animationId,
                  ),
                  nodePositions,
                },
              },
            };
          }),
      }),
      {
        limit: 100,
        partialize: (state) => ({
          record: state.record,
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
  record: EditorRecord,
) {
  store.setState({
    record: structuredClone(record),
    savedRecord: structuredClone(record),
  });
  store.temporal.getState().clear();
}

export function markAssetEditorSessionSaved(
  store: AssetEditorSessionStore,
  record: EditorRecord,
) {
  store.setState({ savedRecord: structuredClone(record) });
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
    case "character.animation.generated":
      store.getState().addGeneratedCharacterAnimation(command.animation);
      return;
    case "character.animation.rename":
      store
        .getState()
        .renameCharacterAnimation(command.animationId, command.label);
      return;
    case "character.animation.delete":
      store.getState().deleteCharacterAnimation(command.animationId);
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
    record: state.record,
    dirty: JSON.stringify(state.record) !== JSON.stringify(state.savedRecord),
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
  const ids = new Set([
    "prototype",
    ...animations.map((animation) => animation.id),
  ]);
  let id = base;
  let suffix = 2;

  while (ids.has(id)) {
    id = `${base}-${suffix}`;
    suffix += 1;
  }

  return id;
}
