import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AssetLibraryItem } from "@/model/asset";
import { withI18n } from "@/testing/with-i18n";

const mocks = vi.hoisted(() => ({
  useRecordQuery: vi.fn(),
}));

vi.mock("@/model/asset", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/model/asset")>()),
  useRecordQuery: mocks.useRecordQuery,
}));

import { AssetPreview } from "./asset-preview";

beforeEach(() => {
  mocks.useRecordQuery.mockReset();
});

describe("AssetPreview", () => {
  it("shows the asset kind fallback when no preview is available", () => {
    expect(renderPreview()).toContain("Preview unavailable");
  });

  it("renders a plain thumbnail", () => {
    const html = renderPreview({ thumbnailUrl: "/object.png" }, "42");

    expect(html).toContain('src="/object.png"');
    expect(html).toContain('alt="Barrel preview"');
  });

  it("applies thumbnail offset and scale", () => {
    const html = renderPreview({
      thumbnailUrl: "/object.png",
      previewOffset: { x: "4px", y: "-2px" },
      previewScale: 1.5,
    });

    expect(html).toContain("translate(4px, -2px) scale(1.5)");
  });

  it("uses default thumbnail transforms when only scale is configured", () => {
    const html = renderPreview({
      thumbnailUrl: "/object.png",
      previewScale: 2,
    });

    expect(html).toContain("translate(0, 0) scale(2)");
  });

  it("uses the default scale when only a thumbnail offset is configured", () => {
    const html = renderPreview({
      thumbnailUrl: "/object.png",
      previewOffset: { x: "1px", y: "2px" },
    });

    expect(html).toContain("translate(1px, 2px) scale(1)");
  });

  it("renders a cropped thumbnail", () => {
    const html = renderPreview({
      thumbnailUrl: "/sheet.png",
      previewCrop: {
        sourceWidth: 128,
        sourceHeight: 64,
        x: 32,
        y: 16,
        width: 32,
        height: 32,
        displayOffsetY: "3px",
      },
    });

    expect(html).toContain("aspect-ratio:32 / 32");
    expect(html).toContain("translateY(3px)");
    expect(html).toContain("height:200%");
    expect(html).toContain("left:-100%");
    expect(html).toContain("top:-50%");
  });

  it("renders a cropped thumbnail without an optional display offset", () => {
    const html = renderPreview({
      thumbnailUrl: "/sheet.png",
      previewCrop: {
        sourceWidth: 64,
        sourceHeight: 64,
        x: 0,
        y: 0,
        width: 32,
        height: 32,
      },
    });

    expect(html).not.toContain("translateY");
  });

  it("renders a selected sprite-sheet frame", () => {
    const html = renderPreview({
      thumbnailUrl: "/sheet.png",
      previewFrame: {
        columns: 4,
        rows: 2,
        column: 1,
        row: 1,
        frameWidth: 32,
        frameHeight: 48,
        displayWidth: "40%",
        offsetX: 5,
      },
    });

    expect(html).toContain("aspect-ratio:32 / 48");
    expect(html).toContain("width:40%");
    expect(html).toContain("height:200%");
    expect(html).toContain("translateX(5px)");
  });

  it("renders a sprite-sheet frame with automatic sizing defaults", () => {
    const html = renderPreview({
      thumbnailUrl: "/sheet.png",
      previewFrame: {
        columns: 2,
        rows: 1,
        column: 0,
        row: 0,
      },
    });

    expect(html).toContain("size-full");
    expect(html).toContain("translateX(0px)");
  });

  it("uses the default width for a sized sprite-sheet frame", () => {
    const html = renderPreview({
      thumbnailUrl: "/sheet.png",
      previewFrame: {
        columns: 1,
        rows: 1,
        column: 0,
        row: 0,
        frameWidth: 32,
        frameHeight: 32,
      },
    });

    expect(html).toContain("width:100%");
  });
});

describe("AssetPreview scenery", () => {
  it("composites visible scenery layers", () => {
    mockRecordReady({
      mode: "scenery",
      prompt: "Misty forest",
      scenery: {
        dimensions: { width: 160, height: 90 },
        layers: [
          {
            id: "sky",
            label: "Sky",
            detail: "Backdrop",
            imageUrl: "/sky.png",
            blendMode: "normal",
            position: { x: 16, y: 9 },
            transform: { scale: { x: 0.5, y: 0.75 }, rotation: 10 },
            opacity: 0.8,
            zIndex: 1,
          },
          {
            id: "trees",
            label: "Trees",
            detail: "Foreground",
            imageUrl: "/trees.png",
            blendMode: "multiply",
            visible: false,
          },
        ],
      },
    });

    const html = renderPreview({ kind: "scenery", name: "Forest" }, "42");

    expect(html).toContain('aria-label="Forest preview"');
    expect(html).toContain('src="/sky.png"');
    expect(html).not.toContain('src="/trees.png"');
    expect(html).toContain("aspect-ratio:160 / 90");
    expect(html).toContain("left:10%");
    expect(html).toContain("top:10%");
    expect(html).toContain("width:50%");
    expect(html).toContain("height:75%");
    expect(html).toContain("opacity:0.8");
    expect(html).toContain("rotate(10deg)");
  });

  it("uses scenery layout defaults", () => {
    mockRecordReady({
      mode: "scenery",
      prompt: "Forest",
      scenery: {
        layers: [
          {
            id: "trees",
            label: "Trees",
            detail: "Foreground",
            imageUrl: "/trees.png",
            blendMode: "multiply",
          },
        ],
      },
    });

    const html = renderPreview({ kind: "scenery" }, "42");

    expect(html).toContain("aspect-ratio:16 / 9");
    expect(html).toContain("mix-blend-multiply");
    expect(html).toContain("left:0%");
    expect(html).toContain("top:0%");
    expect(html).toContain("width:100%");
    expect(html).toContain("height:100%");
    expect(html).toContain("opacity:1");
    expect(html).toContain("rotate(0deg)");
  });

  it("shows a loading state while scenery is fetched", () => {
    mocks.useRecordQuery.mockReturnValue({ status: "pending" });

    expect(renderPreview({ kind: "scenery" }, "42")).toContain("animate-spin");
  });

  it("shows the unavailable state when scenery loading fails", () => {
    mocks.useRecordQuery.mockReturnValue({ status: "error" });

    expect(renderPreview({ kind: "scenery" }, "42")).toContain(
      "Preview unavailable",
    );
  });

  it("shows the unavailable state when scenery data is absent", () => {
    mocks.useRecordQuery.mockReturnValue({
      status: "success",
      data: undefined,
    });

    expect(renderPreview({ kind: "scenery" }, "42")).toContain(
      "Preview unavailable",
    );
  });
});

describe("AssetPreview tileset", () => {
  it("does not intercept the asset card navigation layer", () => {
    mockRecordReady({
      mode: "tileset",
      prompt: "Forest floor",
      tileset: { gridSize: 1, items: [] },
    });

    const html = renderPreview({ kind: "tileset" }, "42");

    expect(html).toMatch(/^<div class="[^"]*pointer-events-none/);
  });

  it("renders individual tiles and complete item images", () => {
    mockRecordReady({
      mode: "tileset",
      prompt: "Forest floor",
      tileset: {
        gridSize: 3,
        items: [
          { id: "empty", label: "Empty", tiles: [] },
          {
            id: "grass",
            label: "Grass",
            tiles: [
              [0, 0],
              [1, 0],
            ],
            tileUrls: ["/grass.png", undefined],
          },
          {
            id: "pond",
            label: "Pond",
            tiles: [
              [1, 1],
              [2, 1],
              [1, 2],
              [2, 2],
            ],
            imageUrl: "/pond.png",
          },
          {
            id: "missing",
            label: "Missing",
            tiles: [[2, 0]],
          },
        ],
      },
    });

    const html = renderPreview({ kind: "tileset" }, "42");

    expect(html).toContain('aria-label="Tileset preview"');
    expect(html).toContain('src="/grass.png"');
    expect(html).toContain('src="/pond.png"');
    expect(html).toContain("grid-column:2 / span 2");
    expect(html).toContain("grid-row:2 / span 2");
    expect(html.match(/aria-hidden="true"/g)).toHaveLength(10);
  });

  it("shows a loading state while tileset data is fetched", () => {
    mocks.useRecordQuery.mockReturnValue({ status: "pending" });

    expect(renderPreview({ kind: "tileset" }, "42")).toContain("animate-spin");
  });

  it("shows the unavailable state when tileset loading fails", () => {
    mocks.useRecordQuery.mockReturnValue({ status: "error" });

    expect(renderPreview({ kind: "tileset" }, "42")).toContain(
      "Preview unavailable",
    );
  });
});

function renderPreview(
  overrides: Partial<AssetLibraryItem> = {},
  projectId?: string,
) {
  return renderToStaticMarkup(
    withI18n(
      <AssetPreview
        asset={{ ...baseAsset, ...overrides }}
        projectId={projectId}
      />,
    ),
  );
}

function mockRecordReady(record: unknown) {
  mocks.useRecordQuery.mockReturnValue({
    status: "success",
    data: { record },
  });
}

const baseAsset: AssetLibraryItem = {
  id: "8",
  kind: "object",
  name: "Barrel",
  description: "Wooden barrel",
  version: "v1",
  canvasSize: "32 × 32 px",
  perspective: "Top-Down",
  tags: [],
  history: [],
  animations: [],
};
