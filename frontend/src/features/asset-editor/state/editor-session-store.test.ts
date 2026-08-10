import { describe, expect, it } from "vitest";

import type { AssetRecord, CharacterAssetRecord } from "@/model";

import {
  createEditorSessionStore,
  dispatchEditorCommand,
  getEditorSessionSnapshot,
  markEditorSessionSaved,
  resetEditorSessionStore,
} from "./editor-session-store";

const idleSaveState = { phase: "idle" } as const;

function createCharacterRecord(): CharacterAssetRecord {
  return {
    mode: "character",
    prompt: "A knight",
    character: {
      prototype: {
        format: "png-sprite-sheet",
        imageUrl: "/knight.png",
        frameWidth: 32,
        frameHeight: 32,
        columns: 1,
        rows: 1,
      },
      animations: [{ kind: "clip", id: "idle", label: "Idle", frameCount: 4 }],
      nodePositions: { idle: { x: 10, y: 20 } },
    },
  };
}

describe("editor session store", () => {
  it("tracks prompt edits through undo, redo, and save baselines", () => {
    const store = createEditorSessionStore(createCharacterRecord());

    dispatchEditorCommand(store, {
      type: "prompt.set",
      value: "A moonlit knight",
    });
    expect(getEditorSessionSnapshot(store, idleSaveState)).toMatchObject({
      record: { prompt: "A moonlit knight" },
      dirty: true,
      canUndo: true,
      canRedo: false,
    });

    dispatchEditorCommand(store, { type: "history.undo" });
    expect(getEditorSessionSnapshot(store, idleSaveState)).toMatchObject({
      record: { prompt: "A knight" },
      dirty: false,
      canUndo: false,
      canRedo: true,
    });

    dispatchEditorCommand(store, { type: "history.redo" });
    markEditorSessionSaved(store, store.getState().record);
    expect(getEditorSessionSnapshot(store, idleSaveState).dirty).toBe(false);
  });

  it("ignores object key order when comparing the saved baseline", () => {
    const record = createCharacterRecord();
    const store = createEditorSessionStore(record);
    const reorderedRecord = {
      prompt: record.prompt,
      character: {
        nodePositions: structuredClone(record.character.nodePositions),
        animations: structuredClone(record.character.animations),
        prototype: structuredClone(record.character.prototype),
      },
      mode: record.mode,
    } satisfies CharacterAssetRecord;

    markEditorSessionSaved(store, reorderedRecord);

    expect(getEditorSessionSnapshot(store, idleSaveState).dirty).toBe(false);
  });

  it("recomputes dirty state after an in-place record mutation", () => {
    const store = createEditorSessionStore(createCharacterRecord());
    const snapshot = getEditorSessionSnapshot(store, idleSaveState);

    snapshot.record.prompt = "Mutated outside the command API";

    expect(getEditorSessionSnapshot(store, idleSaveState).dirty).toBe(true);
  });

  it("records only effective record changes in temporal history", () => {
    const store = createEditorSessionStore(createCharacterRecord());

    dispatchEditorCommand(store, {
      type: "prompt.set",
      value: "A knight",
    });
    dispatchEditorCommand(store, {
      type: "sprite.animation.rename",
      animationId: "missing",
      label: "Run",
    });
    markEditorSessionSaved(store, store.getState().record);

    expect(store.temporal.getState().pastStates).toHaveLength(0);

    dispatchEditorCommand(store, {
      type: "prompt.set",
      value: "A changed knight",
    });
    markEditorSessionSaved(store, store.getState().record);

    expect(store.temporal.getState().pastStates).toHaveLength(1);
  });

  it("adds, renames, and deletes generated character animations", () => {
    const store = createEditorSessionStore(createCharacterRecord());

    dispatchEditorCommand(store, {
      type: "sprite.animation.generated",
      animation: { kind: "clip", label: " Idle ", frameCount: 8 },
    });
    dispatchEditorCommand(store, {
      type: "sprite.animation.rename",
      animationId: "idle-2",
      label: "Run",
    });
    dispatchEditorCommand(store, {
      type: "sprite.node-position.set",
      nodeId: "idle-2",
      position: { x: 30, y: 40 },
    });

    let record = store.getState().record;
    expect(record.mode).toBe("character");
    if (record.mode !== "character") return;
    expect(record.character.animations?.at(-1)).toMatchObject({
      id: "idle-2",
      label: "Run",
    });

    dispatchEditorCommand(store, {
      type: "sprite.animation.delete",
      animationId: "idle-2",
    });
    record = store.getState().record;
    expect(record.mode).toBe("character");
    if (record.mode !== "character") return;
    expect(record.character.animations).toHaveLength(1);
    expect(record.character.nodePositions["idle-2"]).toBeUndefined();
  });

  it("rejects sprite commands for non-sprite asset records", () => {
    const record: AssetRecord = {
      mode: "scenery",
      prompt: "Forest",
      scenery: { layers: [] },
    };
    const store = createEditorSessionStore(record);

    expect(() =>
      dispatchEditorCommand(store, {
        type: "sprite.animation.delete",
        animationId: "idle",
      }),
    ).toThrow("Sprite editing requires a character or object record.");
  });

  it("applies sprite commands to object records", () => {
    const characterRecord = createCharacterRecord();
    const record: AssetRecord = {
      mode: "object",
      prompt: "Crate",
      object: structuredClone(characterRecord.character),
    };
    const store = createEditorSessionStore(record);

    dispatchEditorCommand(store, {
      type: "sprite.animation.delete",
      animationId: "idle",
    });
    expect(store.getState().record).toMatchObject({
      mode: "object",
      object: { animations: [] },
    });
  });

  it("resets the draft, baseline, and temporal history", () => {
    const store = createEditorSessionStore(createCharacterRecord());
    dispatchEditorCommand(store, {
      type: "prompt.set",
      value: "Changed",
    });
    const replacement = createCharacterRecord();
    replacement.prompt = "Restored record";

    resetEditorSessionStore(store, replacement);
    replacement.prompt = "Mutated outside";

    expect(getEditorSessionSnapshot(store, idleSaveState)).toMatchObject({
      record: { prompt: "Restored record" },
      dirty: false,
      canUndo: false,
      canRedo: false,
    });
  });
});
