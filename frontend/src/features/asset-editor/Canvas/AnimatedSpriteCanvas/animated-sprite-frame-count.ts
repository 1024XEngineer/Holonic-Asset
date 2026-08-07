import type { CharacterAnimation, CharacterSpriteSheet } from "@/model";
import { getSpriteSheetFrameCount } from "./sprite-sheet-grid";

type SpriteSheetShape = Pick<CharacterSpriteSheet, "columns" | "rows">;

export function getAnimatedSpriteFrameCount(
  node: string,
  prototype: SpriteSheetShape | undefined,
  animations: readonly CharacterAnimation[] = [],
) {
  if (node === "prototype") return getSpriteSheetFrameCount(prototype);
  return Math.max(
    1,
    animations.find((animation) => animation.id === node)?.frameCount ?? 1,
  );
}
