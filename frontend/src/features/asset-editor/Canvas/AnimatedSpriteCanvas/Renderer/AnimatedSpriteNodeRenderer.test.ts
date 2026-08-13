import { describe, expect, it } from "vitest";

import { isSpriteSheetFrameAvailable } from "./AnimatedSpriteNodeRenderer";

const spriteSheet = {
  format: "png-sprite-sheet" as const,
  imageUrl: "front.png",
  frameUrls: ["front.png", "back.png"],
  frameWidth: 32,
  frameHeight: 32,
  columns: 2,
  rows: 1,
};

describe("isSpriteSheetFrameAvailable", () => {
  it("uses the matching direction URL when checking availability", () => {
    expect(isSpriteSheetFrameAvailable(spriteSheet, 0)).toBe(true);
    expect(
      isSpriteSheetFrameAvailable(spriteSheet, 1, new Set(["back.png"])),
    ).toBe(false);
  });

  it("falls back to the sprite-sheet URL for existing sheets", () => {
    const sheet = { ...spriteSheet, frameUrls: undefined };

    expect(isSpriteSheetFrameAvailable(sheet, 1)).toBe(true);
    expect(isSpriteSheetFrameAvailable(sheet, 1, new Set(["front.png"]))).toBe(
      false,
    );
  });
});
