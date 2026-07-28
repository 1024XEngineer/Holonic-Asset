import { Container, Rectangle, Sprite, Texture } from "pixi.js";

import type { EditorCharacterSpriteSheet } from "../../../domain";

type SpriteSheetFrameBounds = {
  x: number;
  y: number;
  width: number;
  height: number;
};

const spriteSheetFrameTextures = new Map<string, Texture>();

export function drawSpriteSheetFrame({
  container,
  spriteSheet,
  frame,
  bounds,
  pixelScale,
}: {
  container: Container;
  spriteSheet: EditorCharacterSpriteSheet;
  frame: number;
  bounds: SpriteSheetFrameBounds;
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
  container.addChild(sprite);
}

function getSpriteSheetFrameTexture(
  spriteSheet: EditorCharacterSpriteSheet,
  frame: number,
) {
  const column = frame % spriteSheet.columns;
  const row = spriteSheet.row ?? Math.floor(frame / spriteSheet.columns);
  const cacheKey = [
    spriteSheet.imageUrl,
    column,
    row,
    spriteSheet.frameWidth,
    spriteSheet.frameHeight,
  ].join(":");
  let texture = spriteSheetFrameTextures.get(cacheKey);

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
    spriteSheetFrameTextures.set(cacheKey, texture);
  }

  return texture;
}

function snapToPixelGrid(value: number, pixelScale: number) {
  return Math.round(value / pixelScale) * pixelScale;
}
