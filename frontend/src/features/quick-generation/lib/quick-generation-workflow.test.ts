import { describe, expect, it } from "vitest";

import {
  createQuickGenerationDraft,
  defaultQuickGenerationSize,
  toGenerateQuickAssetInput,
} from "./quick-generation-workflow";

describe("quick generation workflow", () => {
  it("creates a blank draft with the default size", () => {
    expect(createQuickGenerationDraft()).toEqual({
      assetId: undefined,
      prompt: "",
      size: defaultQuickGenerationSize,
      creatingReference: undefined,
      creatingReferenceFileName: undefined,
    });
  });

  it("creates an edit draft from an existing asset", () => {
    expect(
      createQuickGenerationDraft(
        {
          id: "asset-1",
          prompt: "Existing prompt",
          size: "128 × 128 px",
          creatingReferenceFileName: "reference.png",
        },
        "blob:reference",
      ),
    ).toEqual({
      assetId: "asset-1",
      prompt: "",
      size: "128 × 128 px",
      creatingReference: "blob:reference",
      creatingReferenceFileName: "reference.png",
    });
  });

  it("normalizes valid input through the Zod schema", () => {
    expect(
      toGenerateQuickAssetInput({
        assetId: "asset-1",
        prompt: "  Moonlit orchard  ",
        size: " 64 x 64 px ",
        creatingReferenceFileName: "reference.png",
      }),
    ).toEqual({
      assetId: "asset-1",
      prompt: "Moonlit orchard",
      size: "64 x 64 px",
      creatingReferenceFileName: "reference.png",
    });
  });

  it("rejects blank required values", () => {
    expect(
      toGenerateQuickAssetInput({ prompt: "   ", size: "64 x 64 px" }),
    ).toBeUndefined();
    expect(
      toGenerateQuickAssetInput({ prompt: "Orchard", size: "   " }),
    ).toBeUndefined();
  });
});
