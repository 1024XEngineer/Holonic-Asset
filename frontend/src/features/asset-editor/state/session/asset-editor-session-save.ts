import type { EditorRecord } from "@/model";

import {
  markAssetEditorSessionSaved,
  type AssetEditorSessionStore,
} from "./asset-editor-session-store";
import type { AssetEditorSaveResult } from "./AssetEditorSession.interface";

type SaveAssetEditorSessionInput = {
  store: AssetEditorSessionStore;
  identity: string;
  isActive: (identity: string) => boolean;
  saveRevision: (record: EditorRecord) => Promise<void>;
};

export async function saveAssetEditorSessionRevision({
  store,
  identity,
  isActive,
  saveRevision,
}: SaveAssetEditorSessionInput): Promise<AssetEditorSaveResult> {
  const submittedRecord = structuredClone(store.getState().record);

  try {
    await saveRevision(submittedRecord);
    if (!isActive(identity)) return { status: "superseded" };

    markAssetEditorSessionSaved(store, submittedRecord);
    return { status: "saved" };
  } catch {
    return isActive(identity) ? { status: "failed" } : { status: "superseded" };
  }
}
