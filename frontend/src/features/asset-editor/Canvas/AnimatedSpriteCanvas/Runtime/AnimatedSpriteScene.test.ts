import { describe, expect, it } from "vitest";
import type { AnimatedSpriteCanvasModel } from "../AnimatedSpriteCanvas.interface";
import {
  AnimatedSpriteScene,
  hasAnimatedSpriteCanvasModelChanged,
} from "./AnimatedSpriteScene";

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
  it("ignores new model wrappers when scene inputs are unchanged", () => {
    const previous = model();
    const next = { ...previous };

    expect(hasAnimatedSpriteCanvasModelChanged(previous, next)).toBe(false);
  });

  it("detects scene input changes", () => {
    const previous = model();
    const next = {
      ...previous,
      selection: { nodeIds: ["prototype"], frames: [] },
    };

    expect(hasAnimatedSpriteCanvasModelChanged(previous, next)).toBe(true);
  });

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
    for (let index = 0; index < 5; index += 1)
      scene.advanceAnimation(model(), 160);
    expect(scene.getSnapshot().previewFrames.get("idle")).toBe(1);
  });

  it("advances simultaneous animations using independent frame timings", () => {
    const timedModel = model({
      animations: [
        {
          kind: "clip",
          id: "fast",
          label: "Fast",
          frameCount: 4,
          frameDurations: [50, 50, 50, 50],
        },
        {
          kind: "clip",
          id: "slow",
          label: "Slow",
          frameCount: 4,
          frameDurations: [200, 200, 200, 200],
        },
      ],
    });
    const scene = new AnimatedSpriteScene(timedModel);
    scene.togglePlaying("fast");
    scene.togglePlaying("slow");

    scene.advanceAnimation(timedModel, 150);

    expect(scene.getSnapshot().previewFrames.get("fast")).toBe(3);
    expect(scene.getSnapshot().previewFrames.get("slow")).toBe(0);

    scene.advanceAnimation(timedModel, 50);

    expect(scene.getSnapshot().previewFrames.get("fast")).toBe(0);
    expect(scene.getSnapshot().previewFrames.get("slow")).toBe(1);
  });

  it("uses per-frame durations and falls back for invalid values", () => {
    const timedModel = model({
      animations: [
        {
          kind: "clip",
          id: "idle",
          label: "Idle",
          frameCount: 3,
          frameDurations: [40, 0, undefined],
        },
      ],
    });
    const scene = new AnimatedSpriteScene(timedModel);
    scene.togglePlaying("idle");

    scene.advanceAnimation(timedModel, 199);
    expect(scene.getSnapshot().previewFrames.get("idle")).toBe(1);

    scene.advanceAnimation(timedModel, 1);
    expect(scene.getSnapshot().previewFrames.get("idle")).toBe(2);
  });
});
