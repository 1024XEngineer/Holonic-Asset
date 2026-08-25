// @vitest-environment happy-dom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { EditPromptForm } from "./edit-prompt-form";
import type { useEditPrompt } from "./use-inspector-edit";

afterEach(cleanup);

function controller(overrides: Record<string, unknown> = {}) {
  return {
    dropzone: {
      getInputProps: () => ({}),
      getRootProps: () => ({}),
      isDragActive: false,
      open: vi.fn(),
    },
    target: null,
    changePrompt: vi.fn(),
    handlePromptKeyDown: vi.fn(),
    handleSubmit: vi.fn((event: Event) => event.preventDefault()),
    creatingReference: null,
    clearCreatingReference: vi.fn(),
    creatingReferenceError: null,
    submitError: null,
    isUploadingCreatingReference: false,
    canSubmit: true,
    ...overrides,
  } as unknown as ReturnType<typeof useEditPrompt>;
}

describe("EditPromptForm", () => {
  it("handles target, prompt, upload, reference, and submit controls", () => {
    const onClearSelection = vi.fn();
    const edit = controller({
      dropzone: {
        getInputProps: () => ({}),
        getRootProps: () => ({}),
        isDragActive: true,
        open: vi.fn(),
      },
      target: {
        label: "Ground",
        detail: "Selected tile",
        thumbnail: {
          imageUrl: "/ground.png",
          column: 0,
          row: 1,
          columns: 2,
          rows: 2,
        },
      },
      creatingReference: {
        fileName: "reference.png",
        mimeType: "image/png",
        objectKey: "uploads/reference.png",
        previewUrl: "/reference.png",
      },
      creatingReferenceError: "Reference failed",
      isUploadingCreatingReference: true,
      isSubmitting: true,
    });

    const view = render(
      withI18n(
        <EditPromptForm
          controller={edit}
          prompt="Add moss"
          isSubmitting
          onClearSelection={onClearSelection}
        />,
      ),
    );

    fireEvent.change(screen.getByLabelText("Edit prompt"), {
      target: { value: "Add moss texture" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Clear selected target" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Attach image" }));
    fireEvent.click(screen.getByRole("button", { name: "Remove image" }));
    fireEvent.submit(view.container.querySelector("form")!);

    expect(edit.changePrompt).toHaveBeenCalledWith("Add moss texture");
    expect(onClearSelection).toHaveBeenCalledOnce();
    expect(edit.dropzone.open).toHaveBeenCalledOnce();
    expect(edit.clearCreatingReference).toHaveBeenCalledOnce();
    expect(edit.handleSubmit).toHaveBeenCalledOnce();
    expect(screen.getByRole("alert").textContent).toContain("Reference failed");
    expect(screen.getByText("Uploading image")).toBeTruthy();
    expect(screen.getByText("Drop image to attach")).toBeTruthy();
  });

  it("renders a target without a thumbnail and the alternate submit error", () => {
    const edit = controller({
      target: { label: "Entire asset", detail: "Whole asset" },
      submitError: "Submit failed",
      canSubmit: false,
    });

    render(
      withI18n(
        <EditPromptForm
          controller={edit}
          prompt=""
          isSubmitting={false}
          onClearSelection={vi.fn()}
        />,
      ),
    );

    expect(screen.getByRole("alert").textContent).toContain("Submit failed");
    expect(
      (screen.getByRole("button", { name: "Send prompt" }) as HTMLButtonElement)
        .disabled,
    ).toBe(true);
  });
});
