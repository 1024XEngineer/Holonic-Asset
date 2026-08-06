import { useCallback, useRef, useState } from "react";
import { useStore } from "zustand";

import { useSaveAssetRevisionMutation } from "@/model";

import { saveEditorSessionRevision } from "./editor-session-save";
import {
  createEditorSessionStore,
  dispatchEditorCommand,
  getEditorSessionSnapshot,
  type EditorSessionStore,
} from "./editor-session-store";
import type {
  EditorCommand,
  EditorSaveState,
  EditorSession,
  UseEditorSessionInput,
} from "./editor-session.types";

type SessionEntry = {
  identity: string;
  store: EditorSessionStore;
};

type SaveStateEntry = {
  store: EditorSessionStore;
  state: EditorSaveState;
};

const idleSaveState: EditorSaveState = { phase: "idle" };

export function useEditorSession({
  target,
  initialRecord,
}: UseEditorSessionInput): EditorSession {
  const identity = `${target.projectId}\0${target.assetId}`;
  const sessionRef = useRef<SessionEntry | null>(null);
  if (sessionRef.current?.identity !== identity) {
    sessionRef.current = {
      identity,
      store: createEditorSessionStore(initialRecord),
    };
  }
  const store = sessionRef.current.store;
  const activeStoreRef = useRef(store);
  activeStoreRef.current = store;
  const latestSaveTokenRef = useRef<symbol | null>(null);
  const [saveStateEntry, setSaveStateEntry] = useState<SaveStateEntry>(() => ({
    store,
    state: idleSaveState,
  }));
  const saveState =
    saveStateEntry.store === store ? saveStateEntry.state : idleSaveState;
  const { mutateAsync: saveRevision } = useSaveAssetRevisionMutation();

  useStore(store, (state) => state.record);
  useStore(store, (state) => state.savedRecord);
  useStore(store.temporal, (state) => state.pastStates.length > 0);
  useStore(store.temporal, (state) => state.futureStates.length > 0);

  const dispatch = useCallback(
    (command: EditorCommand) => {
      dispatchEditorCommand(store, command);
      setSaveStateEntry((current) =>
        current.store === store && current.state.phase === "failed"
          ? { store, state: idleSaveState }
          : current,
      );
    },
    [store],
  );

  const save = useCallback(async () => {
    const saveToken = Symbol("editor-save");
    latestSaveTokenRef.current = saveToken;
    setSaveStateEntry({ store, state: { phase: "saving" } });

    const result = await saveEditorSessionRevision({
      store,
      isActive: () =>
        activeStoreRef.current === store &&
        latestSaveTokenRef.current === saveToken,
      saveRevision: (record) =>
        saveRevision({
          projectId: target.projectId,
          assetId: target.assetId,
          record,
        }).then(() => undefined),
    });

    if (result.status === "saved") {
      setSaveStateEntry({ store, state: idleSaveState });
    } else if (result.status === "failed") {
      setSaveStateEntry({
        store,
        state: { phase: "failed", message: result.message },
      });
    }
    return result;
  }, [saveRevision, store, target.assetId, target.projectId]);

  return {
    snapshot: getEditorSessionSnapshot(store, saveState),
    dispatch,
    save,
  };
}
