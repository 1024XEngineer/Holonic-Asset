import type { EditorSaveState } from "./state";

export type EditorStatusInput = {
  saveState: EditorSaveState;
  isPromptSubmitting: boolean;
  isGeneratingAnimation: boolean;
  notice: string | null;
  isDirty: boolean;
  labels?: Partial<EditorStatusLabels>;
};

export type EditorStatusLabels = {
  saving: string;
  sendingPrompt: string;
  generatingAnimation: string;
  unsavedChanges: string;
  allChangesSaved: string;
};

export function getEditorStatus({
  saveState,
  isPromptSubmitting,
  isGeneratingAnimation,
  notice,
  isDirty,
  labels = {},
}: EditorStatusInput) {
  if (saveState.phase === "saving") return labels.saving ?? "Saving changes";
  if (isPromptSubmitting) return labels.sendingPrompt ?? "Sending prompt";
  if (isGeneratingAnimation)
    return labels.generatingAnimation ?? "Generating animation";
  if (notice) return notice;
  if (saveState.phase === "failed") return saveState.message;

  return isDirty
    ? (labels.unsavedChanges ?? "Unsaved changes")
    : (labels.allChangesSaved ?? "All changes saved");
}
