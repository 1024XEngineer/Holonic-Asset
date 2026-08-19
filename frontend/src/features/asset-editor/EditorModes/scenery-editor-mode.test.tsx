import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import type { AssetWorkspaceDataForKind } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { SceneryEditorMode } from "./scenery-editor-mode";

describe("SceneryEditorMode", () => {
  it("renders the scenery canvas with layer controls and inspection", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <SceneryEditorMode data={workspaceData} onBack={() => undefined} />,
      ),
    );

    expect(html).toContain("scenery editor");
    expect(html).toContain('aria-label="Scenery canvas"');
    expect(html).toContain("Scene layers");
    expect(html).toContain("Inspect");
    expect(html).toContain("Preview ready");
  });
});

const workspaceData: AssetWorkspaceDataForKind<"scenery"> = {
  projectName: "Demo project",
  asset: {
    id: "asset-1",
    projectId: "project-1",
    kind: "scenery",
    name: "Forest",
    perspective: "Top-Down",
    version: "v1",
    history: [],
  },
  record: {
    mode: "scenery",
    prompt: "Forest clearing",
    scenery: {
      layers: [
        {
          id: "sky",
          label: "Sky",
          detail: "Backdrop",
          imageUrl: "/sky.png",
          blendMode: "normal",
          visible: true,
        },
      ],
    },
  },
};
