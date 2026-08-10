import { describe, expect, it } from "vitest";

import { getEditorStatus, type EditorStatusInput } from "./editor-status";

const idleInput: EditorStatusInput = {
  saveState: { phase: "idle" },
  isPromptSubmitting: false,
  isGeneratingAnimation: false,
  notice: null,
  isDirty: false,
};

describe("getEditorStatus", () => {
  it.each([
    ["Saving changes", { saveState: { phase: "saving" } }],
    ["Sending prompt", { isPromptSubmitting: true }],
    ["Generating animation", { isGeneratingAnimation: true }],
    ["Saved just now", { notice: "Saved just now" }],
    ["Save failed", { saveState: { phase: "failed", message: "Save failed" } }],
    ["Unsaved changes", { isDirty: true }],
    ["All changes saved", {}],
  ] satisfies Array<[string, Partial<EditorStatusInput>]>)(
    "returns %s for the matching state",
    (expected, input) => {
      expect(getEditorStatus({ ...idleInput, ...input })).toBe(expected);
    },
  );

  it("keeps the highest-priority status when several states are active", () => {
    expect(
      getEditorStatus({
        ...idleInput,
        saveState: { phase: "saving" },
        isPromptSubmitting: true,
        isGeneratingAnimation: true,
        notice: "Saved just now",
        isDirty: true,
      }),
    ).toBe("Saving changes");
  });
});
