import { describe, expect, it } from "vitest";

import type { CharacterAnimationClip } from "./asset";
import {
  DEFAULT_ANIMATION_FRAME_DURATION_MS,
  getAnimationFrameDuration,
  getEffectiveAnimationFps,
} from "./animation-timing";

const animation = (
  frameDurations?: Array<number | undefined>,
): CharacterAnimationClip => ({
  kind: "clip",
  id: "walk",
  label: "Walk",
  frameCount: frameDurations?.length ?? 2,
  frameDurations,
});

describe("animation timing", () => {
  it("derives effective FPS from backend frame durations", () => {
    expect(getEffectiveAnimationFps(animation([83, 83]))).toBeCloseTo(
      1000 / 83,
    );
  });

  it.each([undefined, 0, -1, Number.NaN, Number.POSITIVE_INFINITY])(
    "uses the documented fallback for an invalid duration: %s",
    (duration) => {
      expect(getAnimationFrameDuration(animation([duration]), 0)).toBe(
        DEFAULT_ANIMATION_FRAME_DURATION_MS,
      );
    },
  );

  it("includes fallback frames when deriving FPS from partial metadata", () => {
    expect(getEffectiveAnimationFps(animation([40, undefined]))).toBe(10);
  });
});
