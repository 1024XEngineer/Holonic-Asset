import { describe, expect, it } from "vitest";
import type { AnimatedSpriteCanvasModel } from "../AnimatedSpriteCanvas.interface";
import { AnimatedSpriteScene } from "./AnimatedSpriteScene";

const model = (
  overrides: Partial<AnimatedSpriteCanvasModel> = {},
): AnimatedSpriteCanvasModel => ({
  prototype: {
    format: "png-sprite-sheet",
    imageUrl: "prototype.png",
    frameWidth: 32,
    frameHeight: 32,
    columns: 4,
    rows: 1,
  },
  animations: [{ kind: "clip", id: "idle", label: "Idle", frameCount: 4 }],
  selection: { nodeIds: [], frames: [] },
  ...overrides,
});

describe("AnimatedSpriteScene", () => {
  it("preserves local positions when model props are synchronized", () => {
    const scene = new AnimatedSpriteScene(
      model({ nodePositions: { idle: { x: 12, y: 18 } } }),
    );
    scene.moveNode("idle", { x: 99, y: 101 });
    scene.synchronize(model());
    expect(scene.getSnapshot().positions.idle).toEqual({ x: 99, y: 101 });
  });

  it("advances playing previews and wraps by frame count", () => {
    const scene = new AnimatedSpriteScene(model());
    scene.togglePlaying("idle");
    for (let index = 0; index < 5; index += 1) scene.advanceAnimation(model());
    expect(scene.getSnapshot().previewFrames.get("idle")).toBe(1);
  });
});
