import { describe, expect, it } from "vitest";

import {
  getRandomAssetTagColor,
  getTagBadgeStyle,
  hexToRgb,
  mergeAssetTags,
  normalizeAssetTag,
  presetAssetTagColors,
} from "./asset-tag";

describe("asset tags", () => {
  it("normalizes legacy names and invalid colors", () => {
    expect(normalizeAssetTag("  hero  ")).toEqual({
      name: "hero",
      description: "",
      color: "#4F46E5",
    });
    expect(
      normalizeAssetTag({ name: "prop", color: "red", description: "  Item " }),
    ).toEqual({
      name: "prop",
      description: "Item",
      color: "#4F46E5",
    });
  });

  it("deduplicates names case-insensitively and preserves rich metadata", () => {
    expect(
      mergeAssetTags(
        [{ name: "Warriors", description: "", color: "#4F46E5" }],
        [
          {
            name: "warriors",
            description: "Combat units",
            color: "#DC2626",
          },
        ],
      ),
    ).toEqual([
      {
        name: "Warriors",
        description: "Combat units",
        color: "#DC2626",
      },
    ]);
  });

  it("handles color helpers and badge styling correctly", () => {
    expect(hexToRgb("#0969DA")).toEqual({ r: 9, g: 105, b: 218 });
    expect(hexToRgb("invalid")).toBeNull();

    const style = getTagBadgeStyle("#0969DA");
    expect(style.backgroundColor).toContain("rgba(9, 105, 218");
    expect(style.borderColor).toContain("rgba(9, 105, 218");

    const fallbackStyle = getTagBadgeStyle("invalid");
    expect(fallbackStyle.backgroundColor).toBeDefined();

    const randomColor = getRandomAssetTagColor("#0969DA");
    expect(presetAssetTagColors).toContain(randomColor);
    expect(randomColor).not.toBe("#0969DA");
  });
});
