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
  } catch (error) {
    return isActive()
      ? { status: "failed", message: getSaveErrorMessage(error) }
      : { status: "superseded" };
  }
}

function getSaveErrorMessage(error: unknown) {
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }

  return "Save failed";
}
