import deepEqual from "fast-deep-equal";
import { createStore } from "zustand";
import { temporal } from "zundo";

import type {
  AssetCanvasPosition,
  AssetRecord,
  SpriteAssetRecordData,
  UISetComponent,
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
  renameSpriteAnimation: (animationId: string, label: string) => void;
  deleteSpriteAnimation: (animationId: string) => void;
  setUISetComponentLabel: (componentId: string, label: string) => void;
  restoreUISetComponent: (component: UISetComponent) => void;
};

const SPRITE_RECORD_REQUIRED_MESSAGE =
  "Sprite editing requires a character or object record.";

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
        setUISetComponentLabel: (componentId, label) =>
          set((state) => {
            const components = getUISetComponents(state.record);
            const target = components.find(
              (component) => component.id === componentId,
            );
            if (!target || target.label === label) {
              return state;
            }

            return {
              record: updateUISetRecord(state.record, (current) => ({
                ...current,
                components: current.components.map((component) =>
                  component.id === componentId
                    ? { ...component, label }
                    : component,
                ),
              })),
            };
          }),
        restoreUISetComponent: (component) =>
          set((state) => {
            const current = getUISetComponents(state.record).find(
              (candidate) => candidate.id === component.id,
            );
            if (!current || deepEqual(current, component)) return state;

            return {
              record: updateUISetRecord(state.record, (currentRecord) => ({
                ...currentRecord,
                components: currentRecord.components.map((candidate) =>
                  candidate.id === component.id
                    ? structuredClone(component)
                    : candidate,
                ),
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

export function syncEditorSessionExternalRecord(
  store: EditorSessionStore,
  incomingRecord: AssetRecord,
) {
  const state = store.getState();
  if (isUISetRecord(state.record) && isUISetRecord(incomingRecord)) {
    syncEditorSessionExternalUISetRecord(store, state, incomingRecord);
    return;
  }
  if (!isSpriteRecord(state.record) || !isSpriteRecord(incomingRecord)) {
    return;
  }
  if (state.record.mode !== incomingRecord.mode) return;

  const incomingSprite = getSpriteRecordData(incomingRecord);
  const record = syncExternalAnimations(
    state.record,
    state.savedRecord,
    incomingRecord,
  );
  const savedRecord = updateSpriteRecord(state.savedRecord, (current) => ({
    ...current,
    animations: structuredClone(incomingSprite.animations ?? []),
  }));
  if (
    recordsMatch(record, state.record) &&
    recordsMatch(savedRecord, state.savedRecord)
  ) {
    return;
  }

  const temporalState = store.temporal.getState();
  const rebaseHistory = (history: typeof temporalState.pastStates) =>
    history.map((snapshot) =>
      snapshot.record
        ? {
            ...snapshot,
            record: syncExternalAnimations(
              snapshot.record,
              state.savedRecord,
              incomingRecord,
            ),
          }
        : snapshot,
    );
  const wasTracking = temporalState.isTracking;
  if (wasTracking) temporalState.pause();
  try {
    store.setState({ record, savedRecord });
    store.temporal.setState({
      pastStates: rebaseHistory(temporalState.pastStates),
      futureStates: rebaseHistory(temporalState.futureStates),
    });
  } finally {
    if (wasTracking) store.temporal.getState().resume();
  }
}

function syncEditorSessionExternalUISetRecord(
  store: EditorSessionStore,
  state: EditorSessionState,
  incomingRecord: Extract<AssetRecord, { mode: "uiset" }>,
) {
  const record = syncExternalUISetRecord(
    state.record,
    state.savedRecord,
    incomingRecord,
  );
  const savedRecord = structuredClone(incomingRecord);
  if (
    recordsMatch(record, state.record) &&
    recordsMatch(savedRecord, state.savedRecord)
  ) {
    return;
  }

  const temporalState = store.temporal.getState();
  const rebaseHistory = (history: typeof temporalState.pastStates) =>
    history.map((snapshot) =>
      snapshot.record
        ? {
            ...snapshot,
            record: syncExternalUISetRecord(
              snapshot.record,
              state.savedRecord,
              incomingRecord,
            ),
          }
        : snapshot,
    );
  const wasTracking = temporalState.isTracking;
  if (wasTracking) temporalState.pause();
  try {
    store.setState({ record, savedRecord });
    store.temporal.setState({
      pastStates: rebaseHistory(temporalState.pastStates),
      futureStates: rebaseHistory(temporalState.futureStates),
    });
  } finally {
    if (wasTracking) store.temporal.getState().resume();
  }
}

function syncExternalAnimations(
  record: AssetRecord,
  savedRecord: AssetRecord,
  incomingRecord: AssetRecord,
) {
  if (
    !isSpriteRecord(record) ||
    !isSpriteRecord(savedRecord) ||
    !isSpriteRecord(incomingRecord) ||
    record.mode !== savedRecord.mode ||
    record.mode !== incomingRecord.mode
  ) {
    return record;
  }
  const currentSprite = getSpriteRecordData(record);
  const savedSprite = getSpriteRecordData(savedRecord);
  const incomingSprite = getSpriteRecordData(incomingRecord);
  return updateSpriteRecord(record, (current) => ({
    ...current,
    animations: mergeExternalAnimations(
      currentSprite.animations ?? [],
      savedSprite.animations ?? [],
      incomingSprite.animations ?? [],
    ),
  }));
}

function syncExternalUISetRecord(
  record: AssetRecord,
  savedRecord: AssetRecord,
  incomingRecord: Extract<AssetRecord, { mode: "uiset" }>,
) {
  if (
    !isUISetRecord(record) ||
    !isUISetRecord(savedRecord) ||
    record.mode !== savedRecord.mode
  ) {
    return record;
  }

  return {
    ...structuredClone(incomingRecord),
    prompt:
      record.prompt === savedRecord.prompt
        ? incomingRecord.prompt
        : record.prompt,
    uiset: {
      ...structuredClone(incomingRecord.uiset),
      components: mergeExternalUISetComponents(
        record.uiset.components,
        savedRecord.uiset.components,
        incomingRecord.uiset.components,
      ),
    },
  };
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
    case "sprite.animation.rename":
      store
        .getState()
        .renameSpriteAnimation(command.animationId, command.label);
      return;
    case "sprite.animation.delete":
      store.getState().deleteSpriteAnimation(command.animationId);
      return;
    case "uiset.component.label.set":
      store
        .getState()
        .setUISetComponentLabel(command.componentId, command.label);
      return;
    case "uiset.component.restore":
      store.getState().restoreUISetComponent(command.component);
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
  throw new Error(SPRITE_RECORD_REQUIRED_MESSAGE);
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
  throw new Error(SPRITE_RECORD_REQUIRED_MESSAGE);
}

function getUISetComponents(record: AssetRecord): UISetComponent[] {
  if (record.mode === "uiset") return record.uiset.components;
  throw new Error("UI Set editing requires a UI Set record.");
}

function updateUISetRecord(
  record: AssetRecord,
  update: (
    data: Extract<AssetRecord, { mode: "uiset" }>["uiset"],
  ) => Extract<AssetRecord, { mode: "uiset" }>["uiset"],
): AssetRecord {
  if (record.mode !== "uiset") {
    throw new Error("UI Set editing requires a UI Set record.");
  }
  return { ...record, uiset: update(record.uiset) };
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

function isSpriteRecord(record: AssetRecord) {
  return record.mode === "character" || record.mode === "object";
}

function isUISetRecord(
  record: AssetRecord,
): record is Extract<AssetRecord, { mode: "uiset" }> {
  return record.mode === "uiset";
}

function mergeExternalAnimations(
  current: SpriteAssetRecordData["animations"],
  saved: SpriteAssetRecordData["animations"],
  incoming: SpriteAssetRecordData["animations"],
) {
  const savedById = new Map(
    saved?.map((animation) => [animation.id, animation]),
  );
  const currentById = new Map(
    current?.map((animation) => [animation.id, animation]),
  );
  const incomingIds = new Set(incoming?.map((animation) => animation.id));
  const merged = (incoming ?? []).flatMap((animation) => {
    const currentAnimation = currentById.get(animation.id);
    const savedAnimation = savedById.get(animation.id);
    if (savedAnimation && !currentAnimation) return [];
    return [
      currentAnimation &&
      savedAnimation &&
      !deepEqual(currentAnimation, savedAnimation)
        ? currentAnimation
        : structuredClone(animation),
    ];
  });

  for (const animation of current ?? []) {
    if (incomingIds.has(animation.id)) continue;
    const savedAnimation = savedById.get(animation.id);
    if (!savedAnimation || !deepEqual(animation, savedAnimation)) {
      merged.push(animation);
    }
  }
  return merged;
}

function mergeExternalUISetComponents(
  current: UISetComponent[],
  saved: UISetComponent[],
  incoming: UISetComponent[],
) {
  const savedById = new Map(
    saved.map((component) => [component.id, component]),
  );
  const currentById = new Map(
    current.map((component) => [component.id, component]),
  );
  const incomingIds = new Set(incoming.map((component) => component.id));
  const merged = incoming.flatMap((component) => {
    const currentComponent = currentById.get(component.id);
    const savedComponent = savedById.get(component.id);
    if (savedComponent && !currentComponent) return [];
    return [
      currentComponent &&
      savedComponent &&
      !deepEqual(currentComponent, savedComponent)
        ? mergeExternalUISetComponent(
            currentComponent,
            savedComponent,
            component,
          )
        : structuredClone(component),
    ];
  });

  for (const component of current) {
    if (incomingIds.has(component.id)) continue;
    const savedComponent = savedById.get(component.id);
    if (!savedComponent || !deepEqual(component, savedComponent)) {
      merged.push(structuredClone(component));
    }
  }
  return merged;
}

function mergeExternalUISetComponent(
  current: UISetComponent,
  saved: UISetComponent,
  incoming: UISetComponent,
) {
  const merged = structuredClone(incoming) as Record<string, unknown>;
  const savedFields = saved as Record<string, unknown>;
  for (const [key, value] of Object.entries(current)) {
    if (!deepEqual(value, savedFields[key])) {
      merged[key] = structuredClone(value);
    }
  }
  return merged as UISetComponent;
}
