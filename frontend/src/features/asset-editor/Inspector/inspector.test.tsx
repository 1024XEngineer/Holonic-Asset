import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import type { CharacterAnimation } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { Inspector } from "./inspector";
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

describe("Inspector", () => {
  it("renders the AI composer controls without the old draft label", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <Inspector
          selectedNodes={[]}
          selectedFrames={[]}
          prompt="Refine the silhouette"
          onPromptChange={() => undefined}
          history={[]}
          animations={animations}
          prototype={prototype}
          onSubmit={() => undefined}
          onClearSelection={() => undefined}
        />,
      ),
    );

    expect(html).not.toContain("Entire asset");
    expect(html).not.toContain("Target");
    expect(html).toContain("Edit");
    expect(html).toContain("What would you like to change?");
    expect(html).toContain("Attach image");
    expect(html).toContain("Send prompt");
    expect(html).not.toContain("Draft context");
  });

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

  it("renders the selected frame thumbnail in the target control", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <Inspector
          selectedNodes={["prototype"]}
          selectedFrames={[{ nodeId: "prototype", index: 1 }]}
          prompt="Refine the silhouette"
          onPromptChange={() => undefined}
          history={[]}
          animations={animations}
          prototype={prototype}
          onSubmit={() => undefined}
          onClearSelection={() => undefined}
        />,
      ),
    );

    expect(html).toContain("Prototype - Frame 2");
    expect(html).toContain('src="/prototype.png"');
    expect(html).toContain("translate(-25%, -0%)");
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
