import deepEqual from "fast-deep-equal";
import { createStore } from "zustand";
import { temporal } from "zundo";

import type {
  AssetCanvasPosition,
  AssetRecord,
  GeneratedCharacterAnimation,
} from "@/model";

import type {
  EditorCommand,
  EditorSaveState,
  EditorSessionSnapshot,
} from "./editor-session.types";

type EditorSessionState = {
  record: AssetRecord;
  savedRecord: AssetRecord;
  setPrompt: (prompt: string) => void;
  setCharacterNodePosition: (
    nodeId: string,
    position: AssetCanvasPosition,
  ) => void;
  addGeneratedCharacterAnimation: (
    animation: GeneratedCharacterAnimation,
  ) => void;
  renameCharacterAnimation: (animationId: string, label: string) => void;
  deleteCharacterAnimation: (animationId: string) => void;
};

export function createEditorSessionStore(initialRecord: AssetRecord) {
  const record = structuredClone(initialRecord);

  return createStore<EditorSessionState>()(
    temporal(
      (set) => ({
        record,
        savedRecord: structuredClone(record),
        setPrompt: (prompt) =>
          set((state) =>
            state.record.prompt === prompt
              ? state
              : { record: { ...state.record, prompt } },
          ),
        setCharacterNodePosition: (nodeId, position) =>
          set((state) => {
            if (state.record.mode !== "character") {
              throw new Error(
                "Character node positions require a character record.",
              );
            }

            const current = state.record.character.nodePositions[nodeId];
            if (
              current &&
              current.x === position.x &&
              current.y === position.y
            ) {
              return state;
            }

            return {
              record: {
                ...state.record,
                character: {
                  ...state.record.character,
                  nodePositions: {
                    ...state.record.character.nodePositions,
                    [nodeId]: { ...position },
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
                      ...structuredClone(animation),
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
            const animations = state.record.character.animations ?? [];
            const target = animations.find(
              (animation) => animation.id === animationId,
            );
            if (
              !normalizedLabel ||
              !target ||
              target.label === normalizedLabel
            ) {
              return state;
            }

            return {
              record: {
                ...state.record,
                character: {
                  ...state.record.character,
                  animations: animations.map((animation) =>
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
            if (!animations.some((animation) => animation.id === animationId)) {
              return state;
            }

            const nodePositions = Object.fromEntries(
              Object.entries(state.record.character.nodePositions).filter(
                ([nodeId]) => nodeId !== animationId,
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
        partialize: (state) => ({ record: state.record }),
        equality: (pastState, currentState) =>
          pastState.record === currentState.record,
      },
    ),
  );
}

export type EditorSessionStore = ReturnType<typeof createEditorSessionStore>;

export function resetEditorSessionStore(
  store: EditorSessionStore,
  record: AssetRecord,
) {
  store.setState({
    record: structuredClone(record),
    savedRecord: structuredClone(record),
  });
  store.temporal.getState().clear();
}

export function markEditorSessionSaved(
  store: EditorSessionStore,
  record: AssetRecord,
) {
  store.setState({ savedRecord: structuredClone(record) });
}

export function dispatchEditorCommand(
  store: EditorSessionStore,
  command: EditorCommand,
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

export function getEditorSessionSnapshot(
  store: EditorSessionStore,
  saveState: EditorSaveState,
): EditorSessionSnapshot {
  const state = store.getState();
  const temporalState = store.temporal.getState();

  return {
    record: state.record,
    dirty: !recordsMatch(state.record, state.savedRecord),
    canUndo: temporalState.pastStates.length > 0,
    canRedo: temporalState.futureStates.length > 0,
    saveState,
  };
}

function recordsMatch(left: AssetRecord, right: AssetRecord) {
  return deepEqual(left, right);
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
