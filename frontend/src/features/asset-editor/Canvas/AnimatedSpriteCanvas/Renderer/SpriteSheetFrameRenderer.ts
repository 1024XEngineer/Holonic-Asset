import { Container, Sprite } from "pixi.js";
import type { CharacterSpriteSheet } from "@/model";
import { snapToStep } from "@/lib/snap-to-step";
import type { SpriteSheetFrameTextureCache } from "./SpriteSheetFrameTextureCache";

type FrameBounds = { x: number; y: number; width: number; height: number };

export function drawSpriteSheetFrame({
  container,
  frameTextures,
  spriteSheet,
  frame,
  bounds,
  pixelScale,
}: {
  container: Container;
  frameTextures: SpriteSheetFrameTextureCache;
  spriteSheet: CharacterSpriteSheet;
  frame: number;
  bounds: FrameBounds;
  pixelScale: number;
}) {
  const texture = frameTextures.get(spriteSheet, frame);
  if (!texture) return;
  const sprite = new Sprite(texture);
  const renderedWidth = spriteSheet.frameWidth * pixelScale;
  const renderedHeight = spriteSheet.frameHeight * pixelScale;
  const centeredX = bounds.x + (bounds.width - renderedWidth) / 2;
  const centeredY = bounds.y + (bounds.height - renderedHeight) / 2;
  sprite.position.set(
    snapToStep(container.x + centeredX, pixelScale) - container.x,
    snapToStep(container.y + centeredY, pixelScale) - container.y,
  );
  sprite.scale.set(pixelScale);
  sprite.texture.source.scaleMode = "nearest";
  container.addChild(sprite);
}
