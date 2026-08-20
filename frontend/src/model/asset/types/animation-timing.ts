import type { CharacterAnimationClip } from "./asset";

/** Legacy and mock animations play at the historical 160 ms per frame. */
export const DEFAULT_ANIMATION_FRAME_DURATION_MS = 160;

export function getAnimationFrameDuration(
  animation: CharacterAnimationClip | undefined,
  frameIndex: number,
) {
  const duration = animation?.frameDurations?.[frameIndex];
  return typeof duration === "number" &&
    Number.isFinite(duration) &&
    duration > 0
    ? duration
    : DEFAULT_ANIMATION_FRAME_DURATION_MS;
}

export function getEffectiveAnimationFps(animation: CharacterAnimationClip) {
  const frameCount = Math.max(1, animation.frameCount);
  let totalDuration = 0;
  for (let frameIndex = 0; frameIndex < frameCount; frameIndex += 1) {
    totalDuration += getAnimationFrameDuration(animation, frameIndex);
  }
  return (frameCount * 1000) / totalDuration;
}
