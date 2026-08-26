// @vitest-environment happy-dom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { AssetWorkspaceData, AssetWorkspaceDataForKind } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { SceneryEditorMode } from "./scenery-editor-mode";

const mocks = vi.hoisted(() => ({
  exportAsset: vi.fn(),
  isExporting: false,
}));

vi.mock("@/model", async (importOriginal) => ({
  ...(await importOriginal()),
  useAssetExport: () => mocks,
}));

beforeEach(() => vi.clearAllMocks());

afterEach(cleanup);

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
    expect(html).toContain('title="Export"');
  });

  it("routes layer selection and visibility events", () => {
    render(
      withI18n(
        <SceneryEditorMode data={workspaceData} onBack={() => undefined} />,
      ),
    );

    const layerButton = screen.getByText("Sky").closest("button");
    if (!layerButton) throw new Error("Expected the scenery layer button.");
    fireEvent.click(layerButton);
    expect(screen.getByText("Selected layer")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Hide Sky" }));
    expect(screen.getByRole("button", { name: "Hidden Toggle" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Hidden Toggle" }));
    expect(screen.getByRole("button", { name: "Hide Sky" })).toBeTruthy();
  });

  it("starts an export for the persisted scenery asset", () => {
    render(
      withI18n(
        <SceneryEditorMode data={workspaceData} onBack={() => undefined} />,
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "Export" }));

    expect(mocks.exportAsset).toHaveBeenCalledWith(1);
  });

  it("does not render for a non-scenery record", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <SceneryEditorMode
          data={tilesetWorkspaceData}
          onBack={() => undefined}
        />,
      ),
    );

    expect(html).toBe("");
  });
});

const workspaceData: AssetWorkspaceDataForKind<"scenery"> = {
  projectName: "Demo project",
  asset: {
    id: "1",
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

const tilesetWorkspaceData: AssetWorkspaceData = {
  projectName: "Demo project",
  asset: {
    ...workspaceData.asset,
    kind: "tileset",
  },
  record: {
    mode: "tileset",
    prompt: "Forest terrain",
    tileset: { gridSize: 16, items: [] },
  },
};
