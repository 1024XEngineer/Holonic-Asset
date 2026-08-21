// @vitest-environment happy-dom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import type { SceneryAssetCreationDraft } from "../types";
import { SceneryAssetFields } from "./scenery-asset-fields";

vi.mock("@/components/ui/custom/image-dropzone", () => ({
  ImageDropzone: ({ onChange }: { onChange: (file: File) => void }) => (
    <button
      type="button"
      onClick={() =>
        onChange(new File(["image"], "orchard.png", { type: "image/png" }))
      }
    >
      Select reference
    </button>
  ),
}));

describe("SceneryAssetFields", () => {
  it("adds the selected creating reference to the scenery draft", () => {
    const onChange = vi.fn();
    const draft: SceneryAssetCreationDraft<File> = {
      kind: "scenery",
      name: "Moonlit orchard",
      prompt: "An orchard under a full moon",
      canvasSize: "1536 × 1024 px",
      aspectRatio: "16:9",
      creatingReference: undefined,
    };
    render(withI18n(<SceneryAssetFields draft={draft} onChange={onChange} />));

    fireEvent.click(screen.getByRole("button", { name: "Select reference" }));

    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({
        ...draft,
        creatingReference: expect.objectContaining({ name: "orchard.png" }),
      }),
    );
  });
});
