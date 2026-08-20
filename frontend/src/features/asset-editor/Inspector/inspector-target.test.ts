import { describe, expect, it } from "vitest";

import type { CharacterAnimation } from "@/model";

import { getInspectorTargetSummary } from "./inspector-target";

const animations: CharacterAnimation[] = [
  {
    kind: "clip",
    id: "idle-front",
    label: "Idle Front",
    frameCount: 4,
    spriteSheet: {
      format: "png-sprite-sheet",
      imageUrl: "/idle-front.png",
      frameWidth: 32,
      frameHeight: 32,
      columns: 2,
      rows: 2,
    },
  },
];

const prototype = {
  format: "png-sprite-sheet" as const,
  imageUrl: "/prototype.png",
  frameWidth: 32,
  frameHeight: 32,
  columns: 4,
  rows: 1,
};

describe("getInspectorTargetSummary", () => {
  it("describes selected animation frames inside the composer", () => {
    expect(
      getInspectorTargetSummary(
        ["idle-front"],
        [
          { nodeId: "idle-front", index: 0 },
          { nodeId: "idle-front", index: 2 },
        ],
        animations,
        prototype,
      ),
    ).toEqual({
      label: "Idle Front - Frames 1, 3",
      detail: "Selected on canvas",
      thumbnail: {
        imageUrl: "/idle-front.png",
        column: 0,
        row: 0,
        columns: 2,
        rows: 2,
      },
    });
  });

  it("names a selected prototype frame in the target control", () => {
    expect(
      getInspectorTargetSummary(
        ["prototype"],
        [{ nodeId: "prototype", index: 0 }],
        animations,
        prototype,
      ),
    ).toEqual({
      label: "Prototype - Frame 1",
      detail: "Selected on canvas",
      thumbnail: {
        imageUrl: "/prototype.png",
        column: 0,
        row: 0,
        columns: 4,
        rows: 1,
      },
    });
  });

  it("uses an independent direction image for the selected prototype frame", () => {
    const directionalPrototype = {
      ...prototype,
      frameUrls: ["/front.png", "/back.png", "/left.png", "/right.png"],
    };

    expect(
      getInspectorTargetSummary(
        ["prototype"],
        [{ nodeId: "prototype", index: 2 }],
        animations,
        directionalPrototype,
      ),
    ).toMatchObject({
      thumbnail: {
        imageUrl: "/left.png",
        column: 0,
        row: 0,
        columns: 1,
        rows: 1,
      },
    });
  });

  it("uses the first independent direction image for a prototype node target", () => {
    const directionalPrototype = {
      ...prototype,
      frameUrls: ["/front.png", "/back.png"],
    };

    expect(
      getInspectorTargetSummary(
        ["prototype"],
        [],
        animations,
        directionalPrototype,
      ),
    ).toMatchObject({
      thumbnail: {
        imageUrl: "/front.png",
        columns: 1,
        rows: 1,
      },
    });
  });

  it("uses the selected animation image for a node target", () => {
    expect(
      getInspectorTargetSummary(["idle-front"], [], animations, prototype),
    ).toEqual({
      label: "Idle Front",
      detail: "Selected item",
      thumbnail: {
        imageUrl: "/idle-front.png",
        column: 0,
        row: 0,
        columns: 2,
        rows: 2,
      },
    });
  });
});
