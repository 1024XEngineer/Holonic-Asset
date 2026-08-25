// @vitest-environment happy-dom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { VisualAssetCreationDraft } from "../types";
import { VisualAssetFields } from "./visual-asset-fields";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/ui/custom/image-dropzone", () => ({
  ImageDropzone: () => null,
}));

describe("VisualAssetFields", () => {
  it("does not publish malformed intermediate custom dimensions", () => {
    const draft: VisualAssetCreationDraft<File> = {
      kind: "character",
      name: "",
      prompt: "",
      canvasSize: "",
      perspective: "Top-Down",
      tags: [],
      creatingReference: undefined,
    };
    const onChange = vi.fn();
    render(<VisualAssetFields draft={draft} onChange={onChange} />);

    fireEvent.change(screen.getByLabelText("canvasWidth"), {
      target: { value: "48" },
    });
    expect(onChange).not.toHaveBeenCalled();

    fireEvent.change(screen.getByLabelText("canvasHeight"), {
      target: { value: "64" },
    });
    expect(onChange).toHaveBeenLastCalledWith({
      ...draft,
      canvasSize: "48 × 64 px",
    });
  });
});
