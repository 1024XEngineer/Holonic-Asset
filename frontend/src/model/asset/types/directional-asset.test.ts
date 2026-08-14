import { describe, expect, it } from "vitest";

import {
  assetDirectionSchema,
  assetDirections,
  assetDirectionsByPerspective,
} from "./directional-asset";

describe("directional asset", () => {
  it("maps each perspective to its prototype direction order", () => {
    expect(assetDirectionsByPerspective).toEqual({
      "Side-On": ["left", "right"],
      "Top-Down": ["front", "right", "back", "left"],
      Isometric: assetDirections,
    });
  });

  it("rejects directions outside the shared asset vocabulary", () => {
    expect(assetDirectionSchema.safeParse("front_left").success).toBe(true);
    expect(assetDirectionSchema.safeParse("up").success).toBe(false);
  });
});
