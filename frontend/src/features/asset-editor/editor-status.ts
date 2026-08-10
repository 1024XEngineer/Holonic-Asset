import type { EditorSaveState } from "./state";

export type EditorStatusInput = {
  saveState: EditorSaveState;
  isPromptSubmitting: boolean;
  isGeneratingAnimation: boolean;
  notice: string | null;
  isDirty: boolean;
};

export function getEditorStatus({
  saveState,
  isPromptSubmitting,
  isGeneratingAnimation,
  notice,
  isDirty,
}: EditorStatusInput) {
  if (saveState.phase === "saving") return "Saving changes";
  if (isPromptSubmitting) return "Sending prompt";
  if (isGeneratingAnimation) return "Generating animation";
  if (notice) return notice;
  if (saveState.phase === "failed") return saveState.message;

  return isDirty ? "Unsaved changes" : "All changes saved";
}
