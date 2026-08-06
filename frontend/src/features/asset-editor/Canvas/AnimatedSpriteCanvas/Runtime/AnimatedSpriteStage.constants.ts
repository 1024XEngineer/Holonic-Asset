import type { CharacterSpriteSheet } from "@/model";

export const MIN_SCALE = 0.3;
export const MAX_SOURCE_PIXEL_SCREEN_SIZE = 24;
export const INITIAL_SCALE = 0.64;
export const NODE_WIDTH = 224;
export const COLLAPSED_HEIGHT = 300;
export const EXPANDED_WIDTH = 448;
export const FRAME_SIZE = 96;
export const FRAME_GAP = 16;
export const PIXEL_GRID_MAJOR_INTERVAL = 8;
export const STAGE_ACCENT = 0xb86b70;
export const STAGE_BACKGROUND = 0xeeece7;

export function getExpandedNodeHeight(frameCount: number) {
  return (
    48 + Math.ceil(Math.max(1, frameCount) / 4) * (FRAME_SIZE + FRAME_GAP) + 48
  );
}

export function getAnimatedSpritePixelScale(
  spriteSheet: Pick<CharacterSpriteSheet, "frameWidth" | "frameHeight">,
) {
  if (spriteSheet.frameWidth <= 0 || spriteSheet.frameHeight <= 0) return 1;
  return Math.min(
    FRAME_SIZE / spriteSheet.frameWidth,
    FRAME_SIZE / spriteSheet.frameHeight,
  );
}

export function getAnimatedSpriteMaxScale(
  spriteSheet: Pick<CharacterSpriteSheet, "frameWidth" | "frameHeight">,
) {
  return (
    MAX_SOURCE_PIXEL_SCREEN_SIZE / getAnimatedSpritePixelScale(spriteSheet)
  );
}
