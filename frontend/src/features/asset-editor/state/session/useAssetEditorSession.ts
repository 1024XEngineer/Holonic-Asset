import { useCallback, useEffect, useRef, useState } from "react";
import { useStore } from "zustand";

import { useSaveAssetRevisionMutation } from "../../api";

import {
  createAssetEditorSessionStore,
  dispatchAssetEditorCommand,
  getAssetEditorSessionSnapshot,
  resetAssetEditorSessionStore,
  type AssetEditorSessionStore,
} from "./asset-editor-session-store";
import { saveAssetEditorSessionRevision } from "./asset-editor-session-save";
import type {
  AssetEditorCommand,
  AssetEditorSaveState,
  AssetEditorSession,
  UseAssetEditorSessionInput,
} from "./AssetEditorSession.interface";

export function useAssetEditorSession({
  target,
  initialRecord,
}: UseAssetEditorSessionInput): AssetEditorSession {
  const storeRef = useRef<AssetEditorSessionStore | null>(null);
  if (storeRef.current === null) {
    storeRef.current = createAssetEditorSessionStore(initialRecord);
  }
  const store = storeRef.current;
  const identity = `${target.projectId}\0${target.assetId}`;
  const activeIdentityRef = useRef(identity);
  activeIdentityRef.current = identity;

  const [saveState, setSaveState] = useState<AssetEditorSaveState>({
    phase: "idle",
  });
  const saveRevisionMutation = useSaveAssetRevisionMutation();
  // These subscriptions keep the React adapter current. Snapshot semantics stay
  // in the session store so production and its tests cross the same seam.
  useStore(store, (state) => state.record);
  useStore(store, (state) => state.savedRecord);
  useStore(store.temporal, (state) => state.pastStates.length > 0);
  useStore(store.temporal, (state) => state.futureStates.length > 0);

  useEffect(() => {
    // Query refreshes for the same target must not overwrite an active draft.
    resetAssetEditorSessionStore(store, initialRecord);
    setSaveState({ phase: "idle" });
  }, [store, target.projectId, target.assetId]);

  const dispatch = useCallback(
    (command: AssetEditorCommand) => {
      dispatchAssetEditorCommand(store, command);
      setSaveState((current) =>
        current.phase === "failed" ? { phase: "idle" } : current,
      );
    },
    [store],
  );

  return {
    snapshot: getAssetEditorSessionSnapshot(store, saveState),
    dispatch,
    save: async () => {
      const submittedIdentity = identity;
      setSaveState({ phase: "saving" });
      const result = await saveAssetEditorSessionRevision({
        store,
        identity: submittedIdentity,
        isActive: (candidate) => activeIdentityRef.current === candidate,
        saveRevision: (record) =>
          saveRevisionMutation
            .mutateAsync({
              projectId: target.projectId,
              assetId: target.assetId,
              record,
            })
            .then(() => undefined),
      });

      if (result.status === "saved") {
        setSaveState({ phase: "idle" });
      } else if (result.status === "failed") {
        setSaveState({ phase: "failed", message: "Save failed" });
      }
      return result;
    },
  };
}
