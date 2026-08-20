// @vitest-environment happy-dom

import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import type { AssetRevision, CharacterAnimation, SceneryLayer } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { Inspector } from "./inspector";

beforeAll(() => {
  Object.defineProperty(Element.prototype, "getAnimations", {
    configurable: true,
    value: () => [],
  });
});

afterEach(cleanup);

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

  it("updates the prompt and previews an attached reference", async () => {
    const onPromptChange = vi.fn();
    render(
      withI18n(
        <Inspector
          kind="sprite"
          selectedNodes={[]}
          selectedFrames={[]}
          prompt="Refine the silhouette"
          onPromptChange={onPromptChange}
          history={[]}
          animations={animations}
          prototype={prototype}
          onSubmit={vi.fn()}
          onClearSelection={vi.fn()}
        />,
      ),
    );

    fireEvent.change(screen.getByLabelText("Edit prompt"), {
      target: { value: "Add a stronger outline" },
    });
    expect(onPromptChange).toHaveBeenCalledWith("Add a stronger outline");

    const fileInput =
      document.querySelector<HTMLInputElement>('input[type="file"]');
    if (!fileInput) throw new Error("Expected the reference file input.");
    fireEvent.change(fileInput, {
      target: {
        files: [
          new File(["reference"], "reference.png", { type: "image/png" }),
        ],
      },
    });

    await waitFor(() =>
      expect(screen.getByAltText("reference.png")).toBeTruthy(),
    );
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

  it("renders scenery defaults when optional layer metadata is absent", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <Inspector
          kind="scenery"
          layer={{
            ...sceneryLayer,
            position: undefined,
            transform: undefined,
            opacity: undefined,
            zIndex: undefined,
          }}
          history={[]}
          visible={false}
          onToggleVisibility={vi.fn()}
        />,
      ),
    );

    expect(html).toContain("Hidden");
    expect(html).toContain("0, 0");
    expect(html).toContain("1.00 x 1.00 / 0°");
    expect(html).toContain("100%");
    expect(html).not.toContain("Canvas:");
  });
});
