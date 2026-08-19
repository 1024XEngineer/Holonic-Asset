import { describe, expect, it } from "vitest";
import type { CharacterAnimation } from "@/model";
import { getAnimatedSpriteFrameCount } from "../animated-sprite-frame-count";
import {
  getAnimatedSpriteNodeLayout,
  getFrameBounds,
  getNodeBounds,
  hitTestAnimatedSpriteScene,
} from "./AnimatedSpriteStageGeometry";

const animations: CharacterAnimation[] = [
  { kind: "clip", id: "idle", label: "Idle", frameCount: 8 },
];

const scene = {
  positions: { prototype: { x: 20, y: 30 }, idle: { x: 300, y: 30 } },
  expanded: new Set(["idle"]),
  playing: new Set<string>(),
  previewFrames: new Map<string, number>(),
  marquee: null,
};

describe("AnimatedSpriteStageGeometry", () => {
  it("supports prototype sheets with multiple frames", () => {
    expect(
      getAnimatedSpriteFrameCount("prototype", { columns: 4, rows: 2 }),
    ).toBe(8);
    expect(
      getNodeBounds(
        "prototype",
        { x: 0, y: 0 },
        true,
        { columns: 4, rows: 2 },
        animations,
      ).height,
    ).toBeGreaterThan(300);
  });

  it("calculates animation bounds from explicit sprite inputs", () => {
    expect(getAnimatedSpriteFrameCount("idle", undefined, animations)).toBe(8);
    expect(
      getNodeBounds(
        "idle",
        { x: 0, y: 0 },
        true,
        { columns: 1, rows: 1 },
        animations,
      ).width,
    ).toBe(448);
  });

  it("hit-tests animation frames before the node body", () => {
    const frame = getFrameBounds(scene.positions.idle, 0);
    expect(
      hitTestAnimatedSpriteScene(
        scene,
        { x: frame.x + 2, y: frame.y + 2 },
        { columns: 1, rows: 1 },
        animations,
      ),
    ).toEqual({ kind: "frame", node: "idle", index: 0 });
  });

  it("places new-animation review controls below play and expand", () => {
    const collapsedScene = { ...scene, expanded: new Set<string>() };
    const review = {
      kind: "new-animation" as const,
      nodeId: "idle",
      isResolving: false,
    };
    const layout = getAnimatedSpriteNodeLayout(
      "idle",
      scene.positions.idle,
      false,
      { columns: 1, rows: 1 },
      animations,
      review,
    );

    expect(layout.bounds.height).toBe(348);
    expect(layout.reviewApplyControl!.y).toBeGreaterThan(layout.playControl!.y);
    expect(
      hitTestAnimatedSpriteScene(
        collapsedScene,
        {
          x: layout.reviewApplyControl!.x + 2,
          y: layout.reviewApplyControl!.y + 2,
        },
        { columns: 1, rows: 1 },
        animations,
        review,
      ),
    ).toEqual({ kind: "review-apply", node: "idle" });
  });

  it("uses one large comparison box with review controls underneath", () => {
    const review = {
      kind: "comparison" as const,
      nodeId: "prototype",
      candidatePrototype: {
        format: "png-sprite-sheet" as const,
        imageUrl: "candidate.png",
        frameWidth: 32,
        frameHeight: 32,
        columns: 4,
        rows: 1,
      },
      isResolving: false,
    };
    const layout = getAnimatedSpriteNodeLayout(
      "prototype",
      scene.positions.prototype,
      false,
      { columns: 4, rows: 1 },
      animations,
      review,
    );

    expect(layout.bounds).toMatchObject({ width: 480, height: 348 });
    expect(layout.frames).toEqual([]);
    expect(layout.reviewApplyControl!.y).toBeGreaterThan(300);
    expect(
      hitTestAnimatedSpriteScene(
        scene,
        {
          x: layout.reviewDenyControl!.x + 2,
          y: layout.reviewDenyControl!.y + 2,
        },
        { columns: 4, rows: 1 },
        animations,
        review,
      ),
    ).toEqual({ kind: "review-deny", node: "prototype" });
  });
});
