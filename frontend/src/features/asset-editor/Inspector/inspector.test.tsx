import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import type { AssetRevision, CharacterAnimation, SceneryLayer } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { Inspector } from "./inspector";

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

const sceneryLayer: SceneryLayer = {
  id: "sky",
  label: "Sky",
  detail: "Backdrop",
  imageUrl: "/sky.png",
  blendMode: "normal",
  position: { x: 0, y: 0 },
  transform: { scale: { x: 1, y: 1 }, rotation: 0 },
  visible: true,
  opacity: 1,
  zIndex: 0,
};

const sceneryHistory: AssetRevision[] = [
  {
    id: "1",
    version: "v1",
    description: "Initial scenery",
    savedAt: "2026-01-01T00:00:00Z",
    status: "ready",
    isCurrent: true,
  },
];

describe("Inspector", () => {
  it("renders the AI composer controls without the old draft label", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <Inspector
          kind="sprite"
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

  it("renders the selected frame thumbnail in the target control", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <Inspector
          kind="sprite"
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

  it("renders scenery content through the shared inspector", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <Inspector
          kind="scenery"
          layer={sceneryLayer}
          dimensions={{ width: 1920, height: 1080 }}
          history={sceneryHistory}
          visible
          onToggleVisibility={vi.fn()}
        />,
      ),
    );

    expect(html).toContain("Selected layer");
    expect(html).toContain("Sky");
    expect(html).toContain("Canvas: 1920 x 1080");
    expect(html).toContain("History");
  });
});
