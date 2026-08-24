import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import type { AssetRecord, AssetWorkspaceData } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { AssetCanvasEditorMode } from "./asset-canvas-editor-mode";

describe("AssetCanvasEditorMode", () => {
  it.each([
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
  ] satisfies Array<
    [AssetWorkspaceData["asset"]["kind"], AssetRecord, string]
  >)(
    "renders the %s record in its editor canvas",
    (kind, record, canvasLabel) => {
      const html = renderToStaticMarkup(
        withI18n(
          <AssetCanvasEditorMode
            data={workspaceData(kind, record)}
            onBack={() => undefined}
          />,
        ),
      );

      expect(html).toContain(`${kind} editor`);
      expect(html).toContain(`aria-label="${canvasLabel}"`);
      expect(html).toContain("Preview ready");
    },
  );
});

function workspaceData(
  kind: AssetWorkspaceData["asset"]["kind"],
  record: AssetRecord,
): AssetWorkspaceData {
  return {
    projectName: "Demo project",
    asset: {
      id: "asset-1",
      projectId: "project-1",
      kind,
      name: "Demo asset",
      perspective: "Top-Down",
      version: "v1",
      history: [],
    },
    record,
  };
}
