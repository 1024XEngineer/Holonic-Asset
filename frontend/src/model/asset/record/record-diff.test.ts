import { describe, expect, it } from "vitest";

import type { CharacterAssetRecord } from "./types";
import { describeAssetRecordChanges } from "./record-diff";

const prototype = {
  format: "png-sprite-sheet" as const,
  imageUrl: "/hero.png",
  frameWidth: 32,
  frameHeight: 32,
  columns: 1,
  rows: 1,
};

function record(
  animations: CharacterAssetRecord["character"]["animations"] = [],
): CharacterAssetRecord {
  return {
    mode: "character",
    prompt: "Hero",
    character: { prototype, animations, nodePositions: {} },
  };
}

describe("describeAssetRecordChanges", () => {
  it("combines prototype and multiple animation changes", () => {
    const before = record();
    const after = record([
      {
        kind: "clip",
        id: "run",
        label: "Run",
        frameCount: 1,
        spriteSheet: { ...prototype, imageUrl: "/run.png" },
      },
      {
        kind: "clip",
        id: "jump",
        label: "Jump",
        frameCount: 1,
        spriteSheet: { ...prototype, imageUrl: "/jump.png" },
      },
    ]);
    after.character.prototype = { ...prototype, imageUrl: "/hero-edited.png" };

    expect(describeAssetRecordChanges(before, after)).toBe(
      "Changed prototype; Added 2 animations: Run, Jump",
    );
  });

  it("reports a rename and an animation content change separately", () => {
    const before = record([
      {
        kind: "clip",
        id: "idle",
        label: "Idle",
        frameCount: 1,
        spriteSheet: { ...prototype, imageUrl: "/idle.png" },
      },
    ]);
    const after = record([
      {
        kind: "clip",
        id: "idle",
        label: "Rest",
        frameCount: 1,
        spriteSheet: { ...prototype, imageUrl: "/rest.png" },
      },
    ]);

    expect(describeAssetRecordChanges(before, after)).toBe(
      "Renamed animation Idle to Rest; Changed animation: Rest",
    );
  });
});
