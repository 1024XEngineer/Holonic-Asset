import { renderToStaticMarkup } from "react-dom/server";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import type { AssetRecord, AssetWorkspaceData } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { AssetCanvasEditorMode } from "./asset-canvas-editor-mode";

vi.mock("@/model/auth", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/model/auth")>()),
  readAuthenticatedUserId: () => 1,
}));

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
      "All changes saved",
      "Asset tree",
    ],
    [
      "uiset",
      {
        mode: "uiset",
        prompt: "Inventory menu",
        uiset: { components: [] },
      },
      "UI Set canvas",
      "Preview ready",
      "UI Set canvas",
    ],
  ] satisfies Array<
    [AssetWorkspaceData["asset"]["kind"], AssetRecord, string, string, string]
  >)(
    "renders the %s record in its editor canvas",
    (kind, record, canvasLabel, status, modeContent) => {
      const html = renderToStaticMarkup(
        withI18n(
          <QueryClientProvider client={new QueryClient()}>
            <AssetCanvasEditorMode
              data={workspaceData(kind, record)}
              onBack={() => undefined}
            />
          </QueryClientProvider>,
        ),
      );

      expect(html).toContain(`${kind} editor`);
      expect(html).toContain(`aria-label="${canvasLabel}"`);
      expect(html).toContain(status);
      expect(html).toContain(modeContent);
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
