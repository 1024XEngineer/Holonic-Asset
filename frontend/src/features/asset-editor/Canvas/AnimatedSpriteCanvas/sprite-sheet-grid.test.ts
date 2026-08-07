import { describe, expect, it } from "vitest";
import {
  getSpriteSheetFrameCount,
  getSpriteSheetFramePosition,
} from "./sprite-sheet-grid";

describe("sprite-sheet-grid", () => {
  it("counts at least one frame using the sheet dimensions", () => {
    expect(getSpriteSheetFrameCount({ columns: 4, rows: 2 })).toBe(8);
    expect(getSpriteSheetFrameCount({ columns: 0, rows: 2 })).toBe(1);
  });

  it("wraps negative and overflowing frames into the sheet grid", () => {
    const grid = { columns: 4, rows: 2 };

    expect(getSpriteSheetFramePosition(-1, grid)).toEqual({
      column: 3,
      row: 1,
    });
    expect(getSpriteSheetFramePosition(8, grid)).toEqual({
      column: 0,
      row: 0,
    });
  });

  it("keeps an explicit source row when one is provided", () => {
    expect(
      getSpriteSheetFramePosition(5, { columns: 4, rows: 2, row: 7 }),
    ).toEqual({ column: 1, row: 7 });
  });
});
