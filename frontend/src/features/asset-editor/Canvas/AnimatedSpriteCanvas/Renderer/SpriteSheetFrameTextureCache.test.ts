import { describe, expect, it, vi } from "vitest";
import type { Texture } from "pixi.js";

import type { CharacterSpriteSheet } from "@/model";

import { SpriteSheetFrameTextureCache } from "./SpriteSheetFrameTextureCache";

function spriteSheet(imageUrl: string): CharacterSpriteSheet {
  return {
    format: "png-sprite-sheet",
    imageUrl,
    frameWidth: 32,
    frameHeight: 32,
    columns: 2,
    rows: 1,
  };
}

describe("SpriteSheetFrameTextureCache", () => {
  it("reuses the texture for the same sprite-sheet frame", () => {
    const createTexture = vi.fn(
      () => ({ destroy: vi.fn() }) as unknown as Texture,
    );
    const cache = new SpriteSheetFrameTextureCache(createTexture);
    const sheet = spriteSheet("idle.png");

    expect(cache.get(sheet, 0)).toBe(cache.get(sheet, 0));
    expect(createTexture).toHaveBeenCalledTimes(1);
  });

  it("releases stale sprite sheets and destroys remaining textures on disposal", () => {
    const created: Array<{ destroy: ReturnType<typeof vi.fn> }> = [];
    const cache = new SpriteSheetFrameTextureCache(() => {
      const texture = { destroy: vi.fn() };
      created.push(texture);
      return texture as unknown as Texture;
    });

    const attack = spriteSheet("attack.png");
    cache.get(spriteSheet("idle.png"), 0);
    cache.get(attack, 0);
    cache.retainSpriteSheets([attack]);

    expect(created[0].destroy).toHaveBeenCalledWith(false);
    expect(created[1].destroy).not.toHaveBeenCalled();

    cache.destroy();

    expect(created[1].destroy).toHaveBeenCalledWith(false);
  });

  it("releases old frame geometry when sprite-sheet metadata changes", () => {
    const created: Array<{ destroy: ReturnType<typeof vi.fn> }> = [];
    const cache = new SpriteSheetFrameTextureCache(() => {
      const texture = { destroy: vi.fn() };
      created.push(texture);
      return texture as unknown as Texture;
    });
    const original = spriteSheet("idle.png");
    const resized = { ...original, frameWidth: 16 };

    cache.get(original, 0);
    cache.get(resized, 0);
    cache.retainSpriteSheets([resized]);

    expect(created[0].destroy).toHaveBeenCalledWith(false);
    expect(created[1].destroy).not.toHaveBeenCalled();
  });

  it("releases old fixed-row textures when the source row changes", () => {
    const created: Array<{ destroy: ReturnType<typeof vi.fn> }> = [];
    const cache = new SpriteSheetFrameTextureCache(() => {
      const texture = { destroy: vi.fn() };
      created.push(texture);
      return texture as unknown as Texture;
    });
    const rowZero = { ...spriteSheet("idle.png"), row: 0 };
    const rowOne = { ...rowZero, row: 1 };

    cache.get(rowZero, 0);
    cache.get(rowOne, 0);
    cache.retainSpriteSheets([rowOne]);

    expect(created[0].destroy).toHaveBeenCalledWith(false);
    expect(created[1].destroy).not.toHaveBeenCalled();
  });
});
