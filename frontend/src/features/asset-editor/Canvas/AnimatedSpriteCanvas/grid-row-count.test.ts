import { describe, expect, it } from "vitest";
import { getGridRowCount } from "./grid-row-count";

describe("getGridRowCount", () => {
  it("returns the rows needed for a fixed column count", () => {
    expect(getGridRowCount(1, 4)).toBe(1);
    expect(getGridRowCount(8, 4)).toBe(2);
    expect(getGridRowCount(9, 4)).toBe(3);
  });

  it("keeps empty grids to one row", () => {
    expect(getGridRowCount(0, 4)).toBe(1);
  });

  it("treats an invalid column count as one column", () => {
    expect(getGridRowCount(3, 0)).toBe(3);
  });
});
