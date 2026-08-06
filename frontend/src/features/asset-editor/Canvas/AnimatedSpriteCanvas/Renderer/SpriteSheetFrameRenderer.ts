import { Container, Rectangle, Sprite, Texture } from "pixi.js";
import type { CharacterSpriteSheet } from "@/model";

type FrameBounds = { x: number; y: number; width: number; height: number };
const frameTextures = new Map<string, Texture>();

export function drawSpriteSheetFrame({
  container,
  spriteSheet,
  frame,
  bounds,
  pixelScale,
}: {
  container: Container;
  spriteSheet: CharacterSpriteSheet;
  frame: number;
  bounds: FrameBounds;
  pixelScale: number;
}) {
  const texture = getSpriteSheetFrameTexture(spriteSheet, frame);
  const sprite = new Sprite(texture);
  const renderedWidth = spriteSheet.frameWidth * pixelScale;
  const renderedHeight = spriteSheet.frameHeight * pixelScale;
  const centeredX = bounds.x + (bounds.width - renderedWidth) / 2;
  const centeredY = bounds.y + (bounds.height - renderedHeight) / 2;
  sprite.position.set(
    snapToPixelGrid(container.x + centeredX, pixelScale) - container.x,
    snapToPixelGrid(container.y + centeredY, pixelScale) - container.y,
  );
  sprite.scale.set(pixelScale);
  sprite.texture.source.scaleMode = "nearest";
  container.addChild(sprite);
}

function getSpriteSheetFrameTexture(
  spriteSheet: CharacterSpriteSheet,
  frame: number,
) {
  const frameCount = Math.max(1, spriteSheet.columns * spriteSheet.rows);
  const safeFrame = ((frame % frameCount) + frameCount) % frameCount;
  const column = safeFrame % Math.max(1, spriteSheet.columns);
  const row =
    spriteSheet.row ?? Math.floor(safeFrame / Math.max(1, spriteSheet.columns));
  const cacheKey = [
    spriteSheet.imageUrl,
    column,
    row,
    spriteSheet.frameWidth,
    spriteSheet.frameHeight,
  ].join(":");
  let texture = frameTextures.get(cacheKey);
  if (!texture) {
    const source = Texture.from(spriteSheet.imageUrl).source;
    texture = new Texture({
      source,
      frame: new Rectangle(
        column * spriteSheet.frameWidth,
        row * spriteSheet.frameHeight,
        spriteSheet.frameWidth,
        spriteSheet.frameHeight,
      ),
    });
    frameTextures.set(cacheKey, texture);
  }
  return texture;
}

function snapToPixelGrid(value: number, pixelScale: number) {
  return pixelScale > 0 ? Math.round(value / pixelScale) * pixelScale : value;
}
