import { Rectangle, Texture } from "pixi.js";

import type { CharacterSpriteSheet } from "@/model";

import { getSpriteSheetFramePosition } from "../sprite-sheet-grid";

type CreateFrameTexture = (
  spriteSheet: CharacterSpriteSheet,
  column: number,
  row: number,
) => Texture;

type FrameTextureEntry = {
  spriteSheetKey: string;
  texture: Texture;
};

export class SpriteSheetFrameTextureCache {
  private readonly textures = new Map<string, FrameTextureEntry>();
  private readonly createTexture: CreateFrameTexture;

  constructor(
    createTexture: CreateFrameTexture = createSpriteSheetFrameTexture,
  ) {
    this.createTexture = createTexture;
  }

  get(spriteSheet: CharacterSpriteSheet, frame: number) {
    const { column, row } = getSpriteSheetFramePosition(frame, spriteSheet);
    const spriteSheetKey = getSpriteSheetKey(spriteSheet);
    const cacheKey = `${spriteSheetKey}:${column}:${row}`;
    let entry = this.textures.get(cacheKey);

    if (!entry) {
      entry = {
        spriteSheetKey,
        texture: this.createTexture(spriteSheet, column, row),
      };
      this.textures.set(cacheKey, entry);
    }

    return entry.texture;
  }

  retainSpriteSheets(spriteSheets: readonly CharacterSpriteSheet[]) {
    const spriteSheetKeys = new Set(spriteSheets.map(getSpriteSheetKey));
    for (const [cacheKey, entry] of this.textures) {
      if (spriteSheetKeys.has(entry.spriteSheetKey)) continue;
      entry.texture.destroy(false);
      this.textures.delete(cacheKey);
    }
  }

  destroy() {
    for (const entry of this.textures.values()) {
      entry.texture.destroy(false);
    }
    this.textures.clear();
  }
}

function getSpriteSheetKey(spriteSheet: CharacterSpriteSheet) {
  return [
    spriteSheet.imageUrl,
    spriteSheet.frameWidth,
    spriteSheet.frameHeight,
    spriteSheet.columns,
    spriteSheet.rows,
    spriteSheet.row,
  ].join(":");
}

function createSpriteSheetFrameTexture(
  spriteSheet: CharacterSpriteSheet,
  column: number,
  row: number,
) {
  return new Texture({
    source: Texture.from(spriteSheet.imageUrl).source,
    frame: new Rectangle(
      column * spriteSheet.frameWidth,
      row * spriteSheet.frameHeight,
      spriteSheet.frameWidth,
      spriteSheet.frameHeight,
    ),
  });
}
