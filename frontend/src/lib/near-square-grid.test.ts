import { describe, expect, it } from "vitest";

import { getNearSquareGrid } from "./near-square-grid";

describe("getNearSquareGrid", () => {
  it.each([
    [0, { columns: 0, rows: 0 }],
    [1, { columns: 1, rows: 1 }],
    [4, { columns: 2, rows: 2 }],
    [5, { columns: 3, rows: 2 }],
    [7, { columns: 3, rows: 3 }],
  ])("returns a near-square grid for %i items", (itemCount, expected) => {
    expect(getNearSquareGrid(itemCount)).toEqual(expected);
  });

  it("normalizes fractional and negative counts", () => {
    expect(getNearSquareGrid(-2)).toEqual({ columns: 0, rows: 0 });
    expect(getNearSquareGrid(5.9)).toEqual({ columns: 3, rows: 2 });
  });
});
