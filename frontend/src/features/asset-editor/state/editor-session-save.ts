import type { AssetRecord } from "@/model";

import {
  markEditorSessionSaved,
  type EditorSessionStore,
} from "./editor-session-store";
import type { EditorSaveResult } from "./editor-session.types";

type SaveEditorSessionInput = {
  store: EditorSessionStore;
  isActive: () => boolean;
  saveRevision: (record: AssetRecord) => Promise<void>;
};

export async function saveEditorSessionRevision({
  store,
  isActive,
  saveRevision,
}: SaveEditorSessionInput): Promise<EditorSaveResult> {
  const submittedRecord = structuredClone(store.getState().record);

  try {
    await saveRevision(submittedRecord);
    if (!isActive()) return { status: "superseded" };

    markEditorSessionSaved(store, submittedRecord);
    return { status: "saved" };
  } catch {
    return isActive() ? { status: "failed" } : { status: "superseded" };
  }
}
