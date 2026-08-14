import { beforeEach, describe, expect, it, vi } from "vitest";

import type { CharacterSpriteSheet } from "@/model";

const mocks = vi.hoisted(() => ({
  sprites: [] as Array<{
    position: { set: ReturnType<typeof vi.fn> };
    scale: { set: ReturnType<typeof vi.fn> };
    texture: { source: { scaleMode: string } };
  }>,
}));

vi.mock("pixi.js", () => ({
  Container: class {},
  Sprite: class {
    position = { set: vi.fn() };
    scale = { set: vi.fn() };
    readonly texture: { source: { scaleMode: string } };

    constructor(texture: { source: { scaleMode: string } }) {
      this.texture = texture;
      mocks.sprites.push(this);
    }
  },
}));

import { drawSpriteSheetFrame } from "./SpriteSheetFrameRenderer";

const spriteSheet: CharacterSpriteSheet = {
  format: "png-sprite-sheet",
  imageUrl: "walk.png",
  frameWidth: 16,
  frameHeight: 16,
  columns: 2,
  rows: 1,
};

describe("drawSpriteSheetFrame", () => {
  beforeEach(() => mocks.sprites.splice(0));

  it("skips a frame whose source texture is not loaded", () => {
    const container = { addChild: vi.fn(), x: 10, y: 20 };
    const get = vi.fn().mockReturnValue(undefined);

    drawSpriteSheetFrame({
      container: container as never,
      frameTextures: { get } as never,
      spriteSheet,
      frame: 1,
      bounds: { x: 5, y: 7, width: 64, height: 64 },
      pixelScale: 2,
    });

    expect(get).toHaveBeenCalledWith(spriteSheet, 1);
    expect(mocks.sprites).toHaveLength(0);
    expect(container.addChild).not.toHaveBeenCalled();
  });

  it("positions and scales an available frame", () => {
    const container = { addChild: vi.fn(), x: 10, y: 20 };
    const texture = { source: { scaleMode: "linear" } };

    drawSpriteSheetFrame({
      container: container as never,
      frameTextures: { get: vi.fn().mockReturnValue(texture) } as never,
      spriteSheet,
      frame: 1,
      bounds: { x: 5, y: 7, width: 64, height: 64 },
      pixelScale: 2,
    });

    const sprite = mocks.sprites[0];
    expect(sprite.position.set).toHaveBeenCalledWith(22, 24);
    expect(sprite.scale.set).toHaveBeenCalledWith(2);
    expect(texture.source.scaleMode).toBe("nearest");
    expect(container.addChild).toHaveBeenCalledWith(sprite);
  });
});
