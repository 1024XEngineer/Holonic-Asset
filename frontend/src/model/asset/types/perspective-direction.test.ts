import { describe, expect, it } from "vitest";

import { getPerspectiveDirectionLayout } from "./perspective-direction";

describe("asset perspective directions", () => {
  it("keeps direction counts and prototype grids aligned", () => {
    expect(getPerspectiveDirectionLayout("Side-On")).toEqual({
      directionCount: 2,
      columns: 2,
      rows: 1,
    });
    expect(getPerspectiveDirectionLayout("Top-Down")).toEqual({
      directionCount: 4,
      columns: 2,
      rows: 2,
    });
    expect(getPerspectiveDirectionLayout("Isometric")).toEqual({
      directionCount: 8,
      columns: 4,
      rows: 2,
    });
  });
});
