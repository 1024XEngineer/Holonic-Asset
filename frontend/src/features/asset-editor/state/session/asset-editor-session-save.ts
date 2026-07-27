import type { RecordContent } from "@/features/assets/domain";

import {
  markAssetEditorSessionSaved,
  type AssetEditorSessionStore,
} from "./asset-editor-session-store";
import type { AssetEditorSaveResult } from "./AssetEditorSession.interface";

type SaveAssetEditorSessionInput = {
  store: AssetEditorSessionStore;
  identity: string;
  isActive: (identity: string) => boolean;
  saveRevision: (document: RecordContent) => Promise<void>;
};

export async function saveAssetEditorSessionRevision({
  store,
  identity,
  isActive,
  saveRevision,
}: SaveAssetEditorSessionInput): Promise<AssetEditorSaveResult> {
  const submittedDocument = structuredClone(store.getState().document);

  try {
    await saveRevision(submittedDocument);
    if (!isActive(identity)) return { status: "superseded" };

    markAssetEditorSessionSaved(store, submittedDocument);
    return { status: "saved" };
  } catch {
    return isActive(identity) ? { status: "failed" } : { status: "superseded" };
  }
}
