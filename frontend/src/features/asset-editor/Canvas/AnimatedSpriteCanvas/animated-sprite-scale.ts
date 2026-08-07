import type { CharacterSpriteSheet } from "@/model";

type SpriteSheetDimensions = Pick<
  CharacterSpriteSheet,
  "frameWidth" | "frameHeight"
>;

export function getAnimatedSpritePixelScale(
  spriteSheet: SpriteSheetDimensions,
  targetSize: number,
) {
  if (spriteSheet.frameWidth <= 0 || spriteSheet.frameHeight <= 0) return 1;
  return Math.min(
    targetSize / spriteSheet.frameWidth,
    targetSize / spriteSheet.frameHeight,
  );
}

export function getAnimatedSpriteMaxScale(
  spriteSheet: SpriteSheetDimensions,
  targetSize: number,
  maxSourcePixelScreenSize: number,
) {
  return (
    maxSourcePixelScreenSize /
    getAnimatedSpritePixelScale(spriteSheet, targetSize)
  );
}
