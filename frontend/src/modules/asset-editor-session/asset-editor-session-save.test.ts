import { describe, expect, it } from "vitest";

import {
  createAssetEditorSessionStore,
  dispatchAssetEditorCommand,
  getAssetEditorSessionSnapshot,
} from "./asset-editor-session-store";
import { saveAssetEditorSessionRevision } from "./asset-editor-session-save";

const idleSaveState = { phase: "idle" } as const;

function createSpriteSheetDocument(prompt: string) {
  return {
    mode: "sprite-sheet" as const,
    prompt,
    spriteSheet: { gridSize: 8, items: [] },
  };
}

describe("asset editor session save", () => {
  it("marks only the submitted snapshot as saved", async () => {
    const store = createAssetEditorSessionStore(
      createSpriteSheetDocument("First edit"),
    );
    let finishSave: (() => void) | undefined;
    const saveRevision = () =>
      new Promise<void>((resolve) => {
        finishSave = resolve;
      });

    const save = saveAssetEditorSessionRevision({
      store,
      identity: "project:asset",
      isActive: () => true,
      saveRevision,
    });
    dispatchAssetEditorCommand(store, {
      type: "prompt.set",
      value: "Edit made during save",
    });
    finishSave?.();

    await expect(save).resolves.toEqual({ status: "saved" });
    expect(getAssetEditorSessionSnapshot(store, idleSaveState)).toMatchObject({
      document: { prompt: "Edit made during save" },
      dirty: true,
    });
  });

  it("preserves the baseline when saving fails", async () => {
    const store = createAssetEditorSessionStore(
      createSpriteSheetDocument("Base prompt"),
    );
    dispatchAssetEditorCommand(store, {
      type: "prompt.set",
      value: "Unsaved prompt",
    });

    await expect(
      saveAssetEditorSessionRevision({
        store,
        identity: "project:asset",
        isActive: () => true,
        saveRevision: () => Promise.reject(new Error("network unavailable")),
      }),
    ).resolves.toEqual({ status: "failed" });
    expect(getAssetEditorSessionSnapshot(store, idleSaveState).dirty).toBe(
      true,
    );
  });

  it("ignores a result for a replaced session", async () => {
    const store = createAssetEditorSessionStore(
      createSpriteSheetDocument("Base prompt"),
    );
    dispatchAssetEditorCommand(store, {
      type: "prompt.set",
      value: "Unsaved prompt",
    });

    await expect(
      saveAssetEditorSessionRevision({
        store,
        identity: "project:old-asset",
        isActive: () => false,
        saveRevision: () => Promise.resolve(),
      }),
    ).resolves.toEqual({ status: "superseded" });
    expect(getAssetEditorSessionSnapshot(store, idleSaveState).dirty).toBe(
      true,
    );
  });
});
