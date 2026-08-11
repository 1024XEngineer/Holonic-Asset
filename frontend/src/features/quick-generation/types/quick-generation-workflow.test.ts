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
      reference: undefined,
      referenceFileName: undefined,
    });
  });

  it("creates an edit draft from an existing asset", () => {
    expect(
      createQuickGenerationDraft(
        {
          id: "asset-1",
          prompt: "Existing prompt",
          size: "128 × 128 px",
          referenceFileName: "reference.png",
        },
        "blob:reference",
      ),
    ).toEqual({
      assetId: "asset-1",
      prompt: "",
      size: "128 × 128 px",
      reference: "blob:reference",
      referenceFileName: "reference.png",
    });
  });

  it("validates and trims generation inputs", () => {
    expect(
      toGenerateQuickAssetInput({ prompt: "  ", size: "64 × 64 px" }),
    ).toBeUndefined();
    expect(
      toGenerateQuickAssetInput({ prompt: "Tree", size: "   " }),
    ).toBeUndefined();
    expect(
      toGenerateQuickAssetInput({
        assetId: "asset-1",
        prompt: "  Ancient tree  ",
        size: " 64 × 64 px ",
        referenceFileName: "tree.png",
      }),
    ).toEqual({
      assetId: "asset-1",
      prompt: "Ancient tree",
      size: "64 × 64 px",
      referenceFileName: "tree.png",
    });
  });
});
