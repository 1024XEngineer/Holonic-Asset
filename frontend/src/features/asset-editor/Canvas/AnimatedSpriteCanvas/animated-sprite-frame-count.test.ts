import { describe, expect, it } from "vitest";
import { getAnimatedSpriteFrameCount } from "./animated-sprite-frame-count";

describe("getAnimatedSpriteFrameCount", () => {
  const animations = [
    { kind: "clip" as const, id: "idle", label: "Idle", frameCount: 8 },
  ];

  it("uses the prototype sheet dimensions for the prototype node", () => {
    expect(
      getAnimatedSpriteFrameCount("prototype", { columns: 4, rows: 2 }),
    ).toBe(8);
  });

  it("uses the matching animation frame count", () => {
    expect(getAnimatedSpriteFrameCount("idle", undefined, animations)).toBe(8);
  });

  it("falls back to one frame for missing or empty animations", () => {
    expect(getAnimatedSpriteFrameCount("missing", undefined, animations)).toBe(
      1,
    );
    expect(
      getAnimatedSpriteFrameCount("empty", undefined, [
        { ...animations[0], id: "empty", frameCount: 0 },
      ]),
    ).toBe(1);
  });
});
