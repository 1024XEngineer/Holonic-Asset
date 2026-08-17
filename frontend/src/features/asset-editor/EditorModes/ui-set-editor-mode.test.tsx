import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import type { AssetWorkspaceData } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { UISetEditorMode } from "./ui-set-editor-mode";

vi.mock("../state", () => ({
  useEditorSession: () => ({
    snapshot: {
      record: workspaceData.record,
      dirty: false,
      canUndo: false,
      canRedo: false,
      saveState: { phase: "idle" },
    },
    dispatch: vi.fn(),
    save: vi.fn(),
    syncExternalRecord: vi.fn(),
  }),
}));

const workspaceData = {
  projectName: "Demo project",
  asset: {
    id: "asset-1",
    projectId: "project-1",
    kind: "uiset",
    name: "Inventory",
    perspective: "Top-Down",
    version: "v1",
    history: [],
  },
  record: {
    mode: "uiset",
    prompt: "Inventory menu",
    uiset: {
      components: [
        {
          id: "panel",
          label: "Panel",
          kind: "panel",
          bounds: { x: 10, y: 10, width: 80, height: 70 },
        },
      ],
    },
  },
} satisfies AssetWorkspaceData;

describe("UISetEditorMode", () => {
  it("renders the component list, canvas, and shared editor panel", () => {
    const html = renderToStaticMarkup(
      withI18n(<UISetEditorMode data={workspaceData} onBack={vi.fn()} />),
    );

    expect(html).toContain("Components");
    expect(html).toContain('aria-label="UI Set canvas"');
    expect(html).toContain("Edit");
    expect(html).toContain("History");
    expect(html).toContain('aria-label="Edit prompt"');
    expect(html).toContain("Panel");
  });
});
