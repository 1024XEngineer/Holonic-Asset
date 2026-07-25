import { describe, expect, it } from "vitest";

import {
  createAssetEditorSessionStore,
  dispatchAssetEditorCommand,
  getAssetEditorSessionSnapshot,
  markAssetEditorSessionSaved,
  resetAssetEditorSessionStore,
} from "./asset-editor-session-store";

const idleSaveState = { phase: "idle" } as const;

function createSpriteSheetDocument(prompt: string) {
  return {
    mode: "sprite-sheet" as const,
    prompt,
    spriteSheet: { gridSize: 8, items: [] },
  };
}

describe("asset editor session commands", () => {
  it("creates an isolated clean snapshot", () => {
    const initialDocument = {
      mode: "character" as const,
      prompt: "Base prompt",
      character: {
        nodePositions: { prototype: { x: 20, y: 40 } },
      },
    };
    const store = createAssetEditorSessionStore(initialDocument);

    initialDocument.character.nodePositions.prototype.x = 999;

    expect(getAssetEditorSessionSnapshot(store, idleSaveState)).toEqual({
      document: {
        mode: "character",
        prompt: "Base prompt",
        character: {
          nodePositions: { prototype: { x: 20, y: 40 } },
        },
      },
      dirty: false,
      canUndo: false,
      canRedo: false,
      saveState: idleSaveState,
    });
  });

  it("applies document commands as one undo step each", () => {
    const store = createAssetEditorSessionStore({
      mode: "character",
      prompt: "Base prompt",
      character: { nodePositions: {} },
    });

    dispatchAssetEditorCommand(store, {
      type: "prompt.set",
      value: "Add a blue scarf",
    });
    dispatchAssetEditorCommand(store, {
      type: "character.node-position.set",
      nodeId: "prototype",
      position: { x: 120, y: 160 },
    });

    expect(getAssetEditorSessionSnapshot(store, idleSaveState)).toMatchObject({
      document: {
        mode: "character",
        prompt: "Add a blue scarf",
        character: {
          nodePositions: { prototype: { x: 120, y: 160 } },
        },
      },
      dirty: true,
      canUndo: true,
      canRedo: false,
    });

    dispatchAssetEditorCommand(store, { type: "history.undo" });
    expect(store.getState().document).toEqual({
      mode: "character",
      prompt: "Add a blue scarf",
      character: { nodePositions: {} },
    });

    dispatchAssetEditorCommand(store, { type: "history.undo" });
    expect(getAssetEditorSessionSnapshot(store, idleSaveState)).toMatchObject({
      document: { prompt: "Base prompt" },
      dirty: false,
      canUndo: false,
      canRedo: true,
    });

    dispatchAssetEditorCommand(store, { type: "history.redo" });
    dispatchAssetEditorCommand(store, { type: "history.redo" });
    expect(store.getState().document).toEqual({
      mode: "character",
      prompt: "Add a blue scarf",
      character: {
        nodePositions: { prototype: { x: 120, y: 160 } },
      },
    });
  });

  it("resets the draft, saved baseline, and history together", () => {
    const store = createAssetEditorSessionStore(
      createSpriteSheetDocument("Base prompt"),
    );
    dispatchAssetEditorCommand(store, {
      type: "prompt.set",
      value: "Temporary edit",
    });

    resetAssetEditorSessionStore(
      store,
      createSpriteSheetDocument("Replacement document"),
    );

    expect(getAssetEditorSessionSnapshot(store, idleSaveState)).toEqual({
      document: createSpriteSheetDocument("Replacement document"),
      dirty: false,
      canUndo: false,
      canRedo: false,
      saveState: idleSaveState,
    });
  });

  it("marks a cloned save baseline without changing history", () => {
    const store = createAssetEditorSessionStore(
      createSpriteSheetDocument("Base prompt"),
    );
    dispatchAssetEditorCommand(store, {
      type: "prompt.set",
      value: "Saved prompt",
    });
    const savedDocument = structuredClone(store.getState().document);

    markAssetEditorSessionSaved(store, savedDocument);
    savedDocument.prompt = "Mutated elsewhere";

    expect(getAssetEditorSessionSnapshot(store, idleSaveState)).toMatchObject({
      document: { prompt: "Saved prompt" },
      dirty: false,
      canUndo: true,
    });
  });
});
