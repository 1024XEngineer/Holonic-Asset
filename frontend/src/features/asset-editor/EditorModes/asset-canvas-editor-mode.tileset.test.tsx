// @vitest-environment happy-dom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

const mocks = vi.hoisted(() => ({
  editor: {
    header: { assetName: "Tileset", status: "saved", onBack: vi.fn() },
    gridSize: 4,
    sourceItems: [
      {
        id: "ground",
        label: "Ground",
        tiles: [[0, 0] as [number, number]],
      },
    ],
    items: [
      {
        id: "ground",
        label: "Ground",
        tiles: [[0, 0] as [number, number]],
      },
    ],
    history: [],
    prompt: "",
    isSubmitting: false,
    review: undefined,
    onPromptChange: vi.fn(),
    onSubmit: vi.fn(),
    onResolveReview: vi.fn(),
  },
  canvas: {
    selectedCellIndexes: [0] as number[],
    selectedItems: ["ground"] as string[],
    selectedLabels: [],
    isCellSelected: vi.fn(() => false),
    send: vi.fn(),
  },
}));

vi.mock("../use-tileset-editor-workspace", () => ({
  useTilesetEditorWorkspace: () => mocks.editor,
}));

vi.mock("../Canvas/TilesetCanvas", () => ({
  TilesetCanvas: ({ onEvent }: { onEvent: (event: unknown) => void }) => (
    <div>
      <button
        type="button"
        onClick={() =>
          onEvent({ type: "cell.selection.toggled", gridCellIndex: 0 })
        }
      >
        canvas select
      </button>
      <button
        type="button"
        onClick={() =>
          onEvent({ type: "generation-review.resolved", applied: true })
        }
      >
        review apply
      </button>
    </div>
  ),
  useTilesetCanvasStateMachine: () => mocks.canvas,
}));

vi.mock("../AssetTree/tileset-asset-tree", () => ({
  TilesetAssetTree: ({
    onToggleItem,
    onToggleTile,
  }: {
    onToggleItem: (itemId: string) => void;
    onToggleTile: (itemId: string, tileIndex: number) => void;
  }) => (
    <div>
      <button type="button" onClick={() => onToggleItem("ground")}>
        item select
      </button>
      <button type="button" onClick={() => onToggleTile("ground", 0)}>
        tile select
      </button>
    </div>
  ),
}));

vi.mock("../Header/editor-header", () => ({
  EditorHeader: () => <header>header</header>,
}));

vi.mock("../Inspector/inspector", () => ({
  Inspector: ({
    onSubmit,
    onClearSelection,
    onPromptChange,
  }: {
    onSubmit: (request: { prompt: string }) => void;
    onClearSelection: () => void;
    onPromptChange: (prompt: string) => void;
  }) => (
    <aside>
      <button type="button" onClick={() => onPromptChange("prompt")}>
        prompt change
      </button>
      <button type="button" onClick={onClearSelection}>
        clear selection
      </button>
      <button type="button" onClick={() => onSubmit({ prompt: "prompt" })}>
        submit edit
      </button>
    </aside>
  ),
}));

import { AssetCanvasEditorMode } from "./asset-canvas-editor-mode";

describe("AssetCanvasEditorMode tileset wiring", () => {
  it("routes tree, canvas, review, and inspector events", () => {
    render(
      withI18n(
        <AssetCanvasEditorMode
          data={{
            projectName: "Project",
            asset: {
              id: "8",
              projectId: "7",
              kind: "tileset",
              name: "Tileset",
              perspective: "Top-Down",
              version: "v1",
              history: [],
            },
            record: {
              mode: "tileset",
              prompt: "",
              tileset: { gridSize: 4, items: mocks.editor.sourceItems },
            },
          }}
          onBack={vi.fn()}
        />,
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "canvas select" }));
    fireEvent.click(screen.getByRole("button", { name: "review apply" }));
    fireEvent.click(screen.getByRole("button", { name: "item select" }));
    fireEvent.click(screen.getByRole("button", { name: "tile select" }));
    fireEvent.click(screen.getByRole("button", { name: "prompt change" }));
    fireEvent.click(screen.getByRole("button", { name: "clear selection" }));
    fireEvent.click(screen.getByRole("button", { name: "submit edit" }));

    expect(mocks.canvas.send).toHaveBeenCalledWith({
      type: "cell.selection.toggled",
      gridCellIndex: 0,
    });
    expect(mocks.canvas.send).toHaveBeenCalledWith({
      type: "item.toggle",
      itemId: "ground",
    });
    expect(mocks.canvas.send).toHaveBeenCalledWith({
      type: "item-cell.toggle",
      itemId: "ground",
      itemCellIndex: 0,
    });
    expect(mocks.canvas.send).toHaveBeenCalledWith({
      type: "selection.cleared",
    });
    expect(mocks.editor.onResolveReview).toHaveBeenCalledWith(true);
    expect(mocks.editor.onSubmit).toHaveBeenCalledWith(
      { prompt: "prompt" },
      expect.objectContaining({ kind: "item", itemId: "ground" }),
    );
  });
});
