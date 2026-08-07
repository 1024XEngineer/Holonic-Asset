import { describe, expect, it } from "vitest";
import { containsPoint, intersectsRect, normalizeRect } from "./rect";

describe("rect", () => {
  it("normalizes points provided in either drag direction", () => {
    expect(normalizeRect({ x: 20, y: 30 }, { x: -5, y: 10 })).toEqual({
      x: -5,
      y: 10,
      width: 25,
      height: 20,
    });
  });

  it("includes points on the rectangle edges", () => {
    const rect = { x: 10, y: 20, width: 30, height: 40 };

    expect(containsPoint(rect, { x: 10, y: 20 })).toBe(true);
    expect(containsPoint(rect, { x: 40, y: 60 })).toBe(true);
    expect(containsPoint(rect, { x: 41, y: 60 })).toBe(false);
  });

  it("does not treat rectangles that only touch as intersecting", () => {
    const rect = { x: 0, y: 0, width: 10, height: 10 };

    expect(intersectsRect(rect, { x: 9, y: 9, width: 2, height: 2 })).toBe(
      true,
    );
    expect(intersectsRect(rect, { x: 10, y: 0, width: 4, height: 4 })).toBe(
      false,
    );
  });
});
