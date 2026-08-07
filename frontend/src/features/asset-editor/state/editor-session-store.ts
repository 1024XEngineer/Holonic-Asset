import deepEqual from "fast-deep-equal";
import { createStore } from "zustand";
import { temporal } from "zundo";

import type {
  AssetCanvasPosition,
  AssetRecord,
  GeneratedCharacterAnimation,
  SpriteAssetRecordData,
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
  setSpriteNodePosition: (
    nodeId: string,
    position: AssetCanvasPosition,
  ) => void;
  addGeneratedSpriteAnimation: (animation: GeneratedCharacterAnimation) => void;
  renameSpriteAnimation: (animationId: string, label: string) => void;
  deleteSpriteAnimation: (animationId: string) => void;
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
        setSpriteNodePosition: (nodeId, position) =>
          set((state) => {
            const sprite = getSpriteRecordData(state.record);
            const current = sprite.nodePositions[nodeId];
            if (
              current &&
              current.x === position.x &&
              current.y === position.y
            ) {
              return state;
            }

            return {
              record: updateSpriteRecord(state.record, (current) => ({
                ...current,
                nodePositions: {
                  ...current.nodePositions,
                  [nodeId]: { ...position },
                },
              })),
            };
          }),
        addGeneratedSpriteAnimation: (animation) =>
          set((state) => {
            const sprite = getSpriteRecordData(state.record);
            const normalizedLabel = animation.label.trim();
            if (!normalizedLabel) return state;

            const animations = sprite.animations ?? [];
            return {
              record: updateSpriteRecord(state.record, (current) => ({
                ...current,
                animations: [
                  ...animations,
                  {
                    ...structuredClone(animation),
                    id: createSpriteAnimationId(normalizedLabel, animations),
                    label: normalizedLabel,
                  },
                ],
              })),
            };
          }),
        renameSpriteAnimation: (animationId, label) =>
          set((state) => {
            const sprite = getSpriteRecordData(state.record);
            const normalizedLabel = label.trim();
            const animations = sprite.animations ?? [];
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
              record: updateSpriteRecord(state.record, (current) => ({
                ...current,
                animations: animations.map((animation) =>
                  animation.id === animationId
                    ? { ...animation, label: normalizedLabel }
                    : animation,
                ),
              })),
            };
          }),
        deleteSpriteAnimation: (animationId) =>
          set((state) => {
            const sprite = getSpriteRecordData(state.record);
            const animations = sprite.animations ?? [];
            if (!animations.some((animation) => animation.id === animationId)) {
              return state;
            }

            const nodePositions = Object.fromEntries(
              Object.entries(sprite.nodePositions).filter(
                ([nodeId]) => nodeId !== animationId,
              ),
            );

            return {
              record: updateSpriteRecord(state.record, (current) => ({
                ...current,
                animations: animations.filter(
                  (animation) => animation.id !== animationId,
                ),
                nodePositions,
              })),
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
    case "sprite.node-position.set":
      store.getState().setSpriteNodePosition(command.nodeId, command.position);
      return;
    case "sprite.animation.generated":
      store.getState().addGeneratedSpriteAnimation(command.animation);
      return;
    case "sprite.animation.rename":
      store
        .getState()
        .renameSpriteAnimation(command.animationId, command.label);
      return;
    case "sprite.animation.delete":
      store.getState().deleteSpriteAnimation(command.animationId);
      return;
    case "history.undo":
      store.temporal.getState().undo();
      return;
    case "history.redo":
      store.temporal.getState().redo();
  }
}

function getSpriteRecordData(record: AssetRecord): SpriteAssetRecordData {
  if (record.mode === "character") return record.character;
  if (record.mode === "object") return record.object;
  throw new Error("Sprite editing requires a character or object record.");
}

function updateSpriteRecord(
  record: AssetRecord,
  update: (data: SpriteAssetRecordData) => SpriteAssetRecordData,
): AssetRecord {
  if (record.mode === "character") {
    return { ...record, character: update(record.character) };
  }
  if (record.mode === "object") {
    return { ...record, object: update(record.object) };
  }
  throw new Error("Sprite editing requires a character or object record.");
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

function createSpriteAnimationId(
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
