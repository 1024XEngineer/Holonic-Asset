import { describe, expect, it } from "vitest";

import { isPerspective, perspectiveOptions } from "./perspective";

describe("perspective", () => {
  it("exposes the canonical contract values", () => {
    expect(perspectiveOptions).toEqual(["Top-Down", "Side-On", "Isometric"]);
  });

  it.each(["TopDown", "Top-down", "top-down", "top_down", "Not specified"])(
    "rejects the non-canonical value %s",
    (value) => {
      expect(isPerspective(value)).toBe(false);
    },
  );
});
