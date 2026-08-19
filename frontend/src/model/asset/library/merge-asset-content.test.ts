import { describe, expect, it } from "vitest";

import { mergeAssetContentPatch } from "./merge-asset-content";

describe("mergeAssetContentPatch", () => {
  it("allocates distinct IDs for generated animations without IDs", () => {
    const merged = mergeAssetContentPatch(
      {
        directionCount: 4,
        prototype: [],
        animations: [
          { id: 4, name: "Idle", frames: [] },
          { id: 7, name: "Walk", frames: [] },
        ],
      },
      {
        animations: [
          { name: "Run", frames: [] },
          { name: "Jump", frames: [] },
        ],
      },
    );

    expect(merged.animations).toEqual([
      { id: 4, name: "Idle", frames: [] },
      { id: 7, name: "Walk", frames: [] },
      { id: 8, name: "Run", frames: [] },
      { id: 9, name: "Jump", frames: [] },
    ]);
  });

  it("preserves IDs for animation edits", () => {
    const merged = mergeAssetContentPatch(
      {
        directionCount: 4,
        prototype: [],
        animations: [{ id: 7, name: "Walk", frames: [{ id: 1 }] }],
      },
      {
        animations: [{ id: 7, name: "Run", frames: [{ id: 2 }] }],
      },
    );

    expect(merged.animations).toEqual([
      { id: 7, name: "Run", frames: [{ id: 2 }] },
    ]);
  });
});
