import { describe, expect, it } from "vitest";

import type { CharacterAssetRecord } from "@/model";

import { saveEditorSessionRevision } from "./editor-session-save";
import {
  createEditorSessionStore,
  dispatchEditorCommand,
  getEditorSessionSnapshot,
} from "./editor-session-store";

const initialRecord: CharacterAssetRecord = {
  mode: "character",
  prompt: "Initial",
  character: {
    prototype: {
      format: "png-sprite-sheet",
      imageUrl: "/character.png",
      frameWidth: 32,
      frameHeight: 32,
      columns: 1,
      rows: 1,
    },
    nodePositions: {},
  },
};

describe("saveEditorSessionRevision", () => {
  it("marks only the submitted snapshot as saved", async () => {
    const store = createEditorSessionStore(initialRecord);
    dispatchEditorCommand(store, {
      type: "prompt.set",
      value: "Submitted",
    });
    let releaseSave!: () => void;
    const pendingSave = new Promise<void>((resolve) => {
      releaseSave = resolve;
    });
    let submittedPrompt = "";

    const resultPromise = saveEditorSessionRevision({
      store,
      isActive: () => true,
      saveRevision: async (record) => {
        submittedPrompt = record.prompt;
        await pendingSave;
      },
    });
    dispatchEditorCommand(store, {
      type: "prompt.set",
      value: "Edited while saving",
    });
    releaseSave();

    await expect(resultPromise).resolves.toEqual({ status: "saved" });
    expect(submittedPrompt).toBe("Submitted");
    expect(getEditorSessionSnapshot(store, { phase: "idle" })).toMatchObject({
      record: { prompt: "Edited while saving" },
      dirty: true,
    });
  });

  it("does not change the save baseline after failure or supersession", async () => {
    const failedStore = createEditorSessionStore(initialRecord);
    dispatchEditorCommand(failedStore, {
      type: "prompt.set",
      value: "Failed draft",
    });

    await expect(
      saveEditorSessionRevision({
        store: failedStore,
        isActive: () => true,
        saveRevision: () => Promise.reject(new Error("network")),
      }),
    ).resolves.toEqual({ status: "failed", message: "network" });
    expect(getEditorSessionSnapshot(failedStore, { phase: "idle" }).dirty).toBe(
      true,
    );

    const supersededStore = createEditorSessionStore(initialRecord);
    dispatchEditorCommand(supersededStore, {
      type: "prompt.set",
      value: "Superseded draft",
    });
    await expect(
      saveEditorSessionRevision({
        store: supersededStore,
        isActive: () => false,
        saveRevision: () => Promise.resolve(),
      }),
    ).resolves.toEqual({ status: "superseded" });
    expect(
      getEditorSessionSnapshot(supersededStore, { phase: "idle" }).dirty,
    ).toBe(true);
  });

  it("uses a generic message when a save rejects with a non-error value", async () => {
    const store = createEditorSessionStore(initialRecord);

    await expect(
      saveEditorSessionRevision({
        store,
        isActive: () => true,
        saveRevision: () => Promise.reject("network"),
      }),
    ).resolves.toEqual({ status: "failed", message: "Save failed" });
  });
});
