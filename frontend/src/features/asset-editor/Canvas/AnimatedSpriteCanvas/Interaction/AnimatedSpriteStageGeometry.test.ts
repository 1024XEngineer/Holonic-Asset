import { describe, expect, it } from "vitest";
import type { CharacterAnimation } from "@/model";
import {
  getFrameBounds,
  getFrameCount,
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
      getFrameCount("prototype", { columns: 4, rows: 2 }, animations),
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

  it("keeps the original animation-only call shape usable", () => {
    expect(getFrameCount("idle", animations)).toBe(8);
    expect(getNodeBounds("idle", { x: 0, y: 0 }, true, animations).width).toBe(
      448,
    );
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
});
