import { describe, expect, it } from "vitest";

import { createDefaultCanvasPositions } from "./AnimatedSpriteCanvas.constants";

function animations(count: number) {
  return Array.from({ length: count }, (_, index) => ({
    kind: "clip" as const,
    id: `animation-${index + 1}`,
    label: `Animation ${index + 1}`,
    frameCount: 1,
  }));
}

describe("createDefaultCanvasPositions", () => {
  it("keeps the prototype and animation order stable", () => {
    const positions = createDefaultCanvasPositions(animations(3));

    expect(positions.prototype.x).toBeLessThan(positions["animation-1"].x);
    expect(positions["animation-1"].y).toBe(positions.prototype.y);
    expect(positions["animation-2"].y).toBeGreaterThan(positions.prototype.y);
    expect(positions["animation-3"].y).toBe(positions["animation-2"].y);
  });
});
