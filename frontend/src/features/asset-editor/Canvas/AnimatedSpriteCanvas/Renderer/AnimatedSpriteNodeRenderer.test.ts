import { describe, expect, it, vi } from "vitest";

import type { CharacterSpriteSheet } from "@/model";

import {
  drawAnimatedSpriteNode,
  getAvailableSpriteSheetFrames,
  isSpriteSheetFrameAvailable,
} from "./AnimatedSpriteNodeRenderer";

vi.mock("pixi.js", () => {
  class Container {
    children: unknown[] = [];
    position = { set: vi.fn() };

    constructor(options: Record<string, unknown> = {}) {
      Object.assign(this, options);
    }

    addChild(child: unknown) {
      this.children.push(child);
      return child;
    }
  }

  class Graphics {
    roundRect() {
      return this;
    }

    rect() {
      return this;
    }

    fill() {
      return this;
    }

    stroke() {
      return this;
    }
  }

  class Text {
    width: number;
    position = { set: vi.fn() };

    constructor(options: { text?: string }) {
      this.width = (options.text?.length ?? 0) * 7;
    }
  }

  return { Container, Graphics, Text };
});

vi.mock("./SpriteSheetFrameRenderer", () => ({
  drawSpriteSheetFrame: vi.fn(),
}));

const { drawSpriteSheetFrame } = await import("./SpriteSheetFrameRenderer");

const prototype: CharacterSpriteSheet = {
  format: "png-sprite-sheet",
  imageUrl: "front.png",
  frameUrls: ["front.png", "back.png"],
  frameWidth: 32,
  frameHeight: 32,
  columns: 2,
  rows: 1,
};

describe("prototype frame rendering", () => {
  it("keeps only available independent direction frames", () => {
    const unavailable = new Set(["back.png"]);

    expect(getAvailableSpriteSheetFrames(prototype, unavailable)).toEqual([0]);
    expect(isSpriteSheetFrameAvailable(prototype, 0, unavailable)).toBe(true);
    expect(isSpriteSheetFrameAvailable(prototype, 1, unavailable)).toBe(false);
  });

  it("renders only available frames in expanded and collapsed previews", () => {
    const drawNode = (expanded: boolean) =>
      drawAnimatedSpriteNode({
        node: "prototype",
        frameTextures: {} as never,
        position: { x: 0, y: 0 },
        selected: false,
        selectedFrames: [],
        expanded,
        playing: false,
        previewFrame: 0,
        animations: [],
        prototype,
        unavailableTextureUrls: new Set(["back.png"]),
      });

    vi.mocked(drawSpriteSheetFrame).mockClear();
    drawNode(true);
    expect(drawSpriteSheetFrame).toHaveBeenCalledTimes(1);
    expect(drawSpriteSheetFrame).toHaveBeenLastCalledWith(
      expect.objectContaining({ frame: 0, spriteSheet: prototype }),
    );

    vi.mocked(drawSpriteSheetFrame).mockClear();
    drawNode(false);
    expect(drawSpriteSheetFrame).toHaveBeenCalledTimes(1);
    expect(drawSpriteSheetFrame).toHaveBeenLastCalledWith(
      expect.objectContaining({ frame: 0, spriteSheet: prototype }),
    );
  });

  it("uses the selected independent frame for a collapsed animation", () => {
    const animation = {
      kind: "clip" as const,
      id: "walk",
      label: "Walk",
      frameCount: 2,
      spriteSheet: {
        ...prototype,
        imageUrl: "walk-1.png",
        frameUrls: ["walk-1.png", "walk-2.png"],
      },
    };
    const drawNode = (previewFrame: number) =>
      drawAnimatedSpriteNode({
        node: "walk",
        frameTextures: {} as never,
        position: { x: 0, y: 0 },
        selected: false,
        selectedFrames: [],
        expanded: false,
        playing: false,
        previewFrame,
        animations: [animation],
        prototype,
        unavailableTextureUrls: new Set(["walk-2.png"]),
      });

    vi.mocked(drawSpriteSheetFrame).mockClear();
    drawNode(1);
    expect(drawSpriteSheetFrame).not.toHaveBeenCalled();

    drawNode(0);
    expect(drawSpriteSheetFrame).toHaveBeenCalledWith(
      expect.objectContaining({
        frame: 0,
        spriteSheet: animation.spriteSheet,
      }),
    );
  });
});
