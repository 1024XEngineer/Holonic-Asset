import type { CharacterSpriteSheet } from "@/model";

export type SpriteSheetGrid = Pick<CharacterSpriteSheet, "columns" | "rows"> & {
  row?: number;
};

export function getSpriteSheetFrameCount(
  spriteSheet?: Pick<CharacterSpriteSheet, "columns" | "rows">,
) {
  return Math.max(1, (spriteSheet?.columns ?? 1) * (spriteSheet?.rows ?? 1));
}

export function getSpriteSheetFramePosition(
  frame: number,
  spriteSheet: SpriteSheetGrid,
) {
  const frameCount = getSpriteSheetFrameCount(spriteSheet);
  const safeFrame = ((frame % frameCount) + frameCount) % frameCount;
  const columns = Math.max(1, spriteSheet.columns);

  return {
    column: safeFrame % columns,
    row: spriteSheet.row ?? Math.floor(safeFrame / columns),
  };
}
