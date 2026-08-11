import { describe, expect, it } from "vitest";

import { toGenerateQuickAssetInput } from "./quick-generation-workflow";

describe("toGenerateQuickAssetInput", () => {
  it("normalizes valid input through the Zod schema", () => {
    expect(
      toGenerateQuickAssetInput({
        assetId: "asset-1",
        prompt: "  Moonlit orchard  ",
        size: " 64 x 64 px ",
        referenceFileName: "reference.png",
      }),
    ).toEqual({
      assetId: "asset-1",
      prompt: "Moonlit orchard",
      size: "64 x 64 px",
      referenceFileName: "reference.png",
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
