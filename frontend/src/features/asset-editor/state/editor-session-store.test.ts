import { describe, expect, it } from "vitest";

import type {
  AssetRecord,
  CharacterAssetRecord,
  UISetAssetRecord,
} from "@/model";

import {
  createEditorSessionStore,
  dispatchEditorCommand,
  getEditorSessionSnapshot,
  markEditorSessionSaved,
  resetEditorSessionStore,
  syncEditorSessionExternalRecord,
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

function createUISetRecord(): UISetAssetRecord {
  return {
    mode: "uiset",
    prompt: "Inventory menu",
    uiset: {
      components: [
        {
          id: "primary-action",
          label: "Start",
          kind: "button",
          bounds: { x: 20, y: 60, width: 50, height: 16 },
        },
      ],
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

  it("renames and deletes character animations", () => {
    const store = createEditorSessionStore(createCharacterRecord());

    dispatchEditorCommand(store, {
      type: "sprite.animation.rename",
      animationId: "idle",
      label: "Run",
    });
    dispatchEditorCommand(store, {
      type: "sprite.node-position.set",
      nodeId: "idle",
      position: { x: 30, y: 40 },
    });

    let record = store.getState().record;
    expect(record.mode).toBe("character");
    if (record.mode !== "character") return;
    expect(record.character.animations?.at(-1)).toMatchObject({
      id: "idle",
      label: "Run",
    });

    dispatchEditorCommand(store, {
      type: "sprite.animation.delete",
      animationId: "idle",
    });
    record = store.getState().record;
    expect(record.mode).toBe("character");
    if (record.mode !== "character") return;
    expect(record.character.animations).toHaveLength(0);
    expect(record.character.nodePositions.idle).toBeUndefined();
  });

  it("updates and restores UI Set components through editor history", () => {
    const generatedRecord = createUISetRecord();
    const store = createEditorSessionStore(generatedRecord);
    const generatedComponent = generatedRecord.uiset.components[0];
    if (!generatedComponent) throw new Error("Expected a generated component.");

    dispatchEditorCommand(store, {
      type: "uiset.component.label.set",
      componentId: generatedComponent.id,
      label: "Play now",
    });
    expect(store.getState().record).toMatchObject({
      mode: "uiset",
      uiset: { components: [{ id: "primary-action", label: "Play now" }] },
    });

    dispatchEditorCommand(store, {
      type: "uiset.component.restore",
      component: generatedComponent,
    });
    expect(getEditorSessionSnapshot(store, idleSaveState)).toMatchObject({
      record: {
        mode: "uiset",
        uiset: { components: [{ id: "primary-action", label: "Start" }] },
      },
      dirty: false,
      canUndo: true,
    });

    dispatchEditorCommand(store, { type: "history.undo" });
    expect(store.getState().record).toMatchObject({
      mode: "uiset",
      uiset: { components: [{ id: "primary-action", label: "Play now" }] },
    });
  });

  it("ignores no-op UI Set edits and rejects UI Set commands for other assets", () => {
    const store = createEditorSessionStore(createUISetRecord());
    const record = store.getState().record;
    if (record.mode !== "uiset") throw new Error("Expected a UI Set record.");
    const component = record.uiset.components[0];
    if (!component) throw new Error("Expected a UI Set component.");

    dispatchEditorCommand(store, {
      type: "uiset.component.label.set",
      componentId: component.id,
      label: "   ",
    });
    dispatchEditorCommand(store, {
      type: "uiset.component.label.set",
      componentId: "missing",
      label: "Rename",
    });
    dispatchEditorCommand(store, {
      type: "uiset.component.restore",
      component,
    });
    expect(store.temporal.getState().pastStates).toHaveLength(0);

    const sceneryStore = createEditorSessionStore({
      mode: "scenery",
      prompt: "Forest",
      scenery: { layers: [] },
    });
    expect(() =>
      dispatchEditorCommand(sceneryStore, {
        type: "uiset.component.label.set",
        componentId: "panel",
        label: "Panel",
      }),
    ).toThrow("UI Set editing requires a UI Set record.");
  });

  it("merges generated server animations without overwriting local edits", () => {
    const store = createEditorSessionStore(createCharacterRecord());
    dispatchEditorCommand(store, {
      type: "sprite.animation.rename",
      animationId: "idle",
      label: "Local idle edit",
    });
    const incoming = createCharacterRecord();
    incoming.character.animations = [
      { kind: "clip", id: "idle", label: "Server idle", frameCount: 4 },
      { kind: "clip", id: "walk", label: "Walk", frameCount: 8 },
    ];

    syncEditorSessionExternalRecord(store, incoming);

    const snapshot = getEditorSessionSnapshot(store, idleSaveState);
    expect(snapshot.record).toMatchObject({
      mode: "character",
      character: {
        animations: [
          { id: "idle", label: "Local idle edit" },
          { id: "walk", label: "Walk" },
        ],
      },
    });
    expect(snapshot.dirty).toBe(true);
  });

  it("keeps external synchronization out of undo history", () => {
    const initialRecord = createCharacterRecord();
    const store = createEditorSessionStore(initialRecord);

    syncEditorSessionExternalRecord(store, structuredClone(initialRecord));

    expect(store.temporal.getState().pastStates).toHaveLength(0);

    dispatchEditorCommand(store, {
      type: "sprite.animation.rename",
      animationId: "idle",
      label: "Local idle edit",
    });
    const incoming = createCharacterRecord();
    incoming.character.animations = [
      { kind: "clip", id: "idle", label: "Server idle", frameCount: 4 },
      { kind: "clip", id: "walk", label: "Walk", frameCount: 8 },
    ];

    syncEditorSessionExternalRecord(store, incoming);

    expect(store.temporal.getState().pastStates).toHaveLength(1);
    dispatchEditorCommand(store, { type: "history.undo" });
    expect(getEditorSessionSnapshot(store, idleSaveState)).toMatchObject({
      record: {
        mode: "character",
        character: {
          animations: [
            { id: "idle", label: "Server idle" },
            { id: "walk", label: "Walk" },
          ],
        },
      },
      dirty: false,
    });
  });

  it("does not restore a locally deleted animation during a server refresh", () => {
    const store = createEditorSessionStore(createCharacterRecord());
    dispatchEditorCommand(store, {
      type: "sprite.animation.delete",
      animationId: "idle",
    });
    const incoming = createCharacterRecord();
    incoming.character.animations?.push({
      kind: "clip",
      id: "walk",
      label: "Walk",
      frameCount: 8,
    });

    syncEditorSessionExternalRecord(store, incoming);

    expect(store.getState().record).toMatchObject({
      mode: "character",
      character: { animations: [{ id: "walk" }] },
    });
    expect(getEditorSessionSnapshot(store, idleSaveState).dirty).toBe(true);
  });

  it("preserves a locally edited animation removed by a server refresh", () => {
    const store = createEditorSessionStore(createCharacterRecord());
    dispatchEditorCommand(store, {
      type: "sprite.animation.rename",
      animationId: "idle",
      label: "Local idle edit",
    });
    const incoming = createCharacterRecord();
    incoming.character.animations = [];

    syncEditorSessionExternalRecord(store, incoming);

    expect(store.getState().record).toMatchObject({
      mode: "character",
      character: {
        animations: [{ id: "idle", label: "Local idle edit" }],
      },
    });
    expect(getEditorSessionSnapshot(store, idleSaveState).dirty).toBe(true);
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
