import { describe, expect, it } from "vitest";
import {
  getAnimatedSpriteMaxScale,
  getAnimatedSpritePixelScale,
} from "./animated-sprite-scale";

describe("animated-sprite-scale", () => {
  it("fits both source dimensions inside the target size", () => {
    expect(
      getAnimatedSpritePixelScale({ frameWidth: 32, frameHeight: 16 }, 96),
    ).toBe(3);
  });

  it("uses a neutral scale for invalid source dimensions", () => {
    expect(
      getAnimatedSpritePixelScale({ frameWidth: 0, frameHeight: 32 }, 96),
    ).toBe(1);
  });

  it("derives the maximum source scale from the pixel display limit", () => {
    expect(
      getAnimatedSpriteMaxScale({ frameWidth: 32, frameHeight: 32 }, 96, 24),
    ).toBe(8);
  });
});
