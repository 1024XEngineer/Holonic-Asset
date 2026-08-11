import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import type { AssetKind, AssetRecord, AssetWorkspaceData } from "@/model";

import { AssetCanvasEditorMode } from "./asset-canvas-editor-mode";

describe("AssetCanvasEditorMode", () => {
  it.each([
    [
      "scenery",
      {
        mode: "scenery",
        prompt: "Forest clearing",
        scenery: { layers: [] },
      },
      "Scenery canvas",
    ],
    [
      "tileset",
      {
        mode: "tileset",
        prompt: "Stone floor",
        tileset: { gridSize: 8, items: [] },
      },
      "Tileset canvas",
    ],
    [
      "uiset",
      {
        mode: "uiset",
        prompt: "Inventory menu",
        uiset: { components: [] },
      },
      "UI Set canvas",
    ],
  ] satisfies Array<[AssetKind, AssetRecord, string]>)(
    "renders the %s record in its editor canvas",
    (kind, record, canvasLabel) => {
      const html = renderToStaticMarkup(
        <AssetCanvasEditorMode
          data={workspaceData(kind, record)}
          onBack={() => undefined}
        />,
      );

      expect(html).toContain(`${kind} editor`);
      expect(html).toContain(`aria-label="${canvasLabel}"`);
      expect(html).toContain("Preview ready");
    },
  );

  it("renders no canvas for unsupported record modes", () => {
    const html = renderToStaticMarkup(
      <AssetCanvasEditorMode
        data={workspaceData("audio", {
          mode: "audio",
          prompt: "Theme",
          audio: {},
        })}
        onBack={() => undefined}
      />,
    );

    expect(html).toBe("");
  });
});

function workspaceData(
  kind: AssetKind,
  record: AssetRecord,
): AssetWorkspaceData {
  return {
    projectName: "Demo project",
    asset: {
      id: "asset-1",
      projectId: "project-1",
      kind,
      name: "Demo asset",
      version: "v1",
      history: [],
    },
    record,
  };
}
