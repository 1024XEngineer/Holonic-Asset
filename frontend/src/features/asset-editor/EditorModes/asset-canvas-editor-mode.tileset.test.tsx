// @vitest-environment happy-dom

import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

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
    review: undefined as { items: never[]; isResolving: boolean } | undefined,
    onPromptChange: vi.fn(),
    onSubmit: vi.fn(),
    onResolveReview: vi.fn(),
  },
  editorEnabled: true,
  canvas: {
    selectedCellIndexes: [0] as number[],
    selectedItems: ["ground"] as string[],
    selectedLabels: [],
    isCellSelected: vi.fn(() => false),
    send: vi.fn(),
  },
}));

vi.mock("../use-tileset-editor-workspace", () => ({
  useTilesetEditorWorkspace: () => (mocks.editorEnabled ? mocks.editor : null),
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
    targetError,
  }: {
    onSubmit: (request: { prompt: string }) => void;
    onClearSelection: () => void;
    onPromptChange: (prompt: string) => void;
    targetError: string | null;
  }) => (
    <aside>
      {targetError ? <p>{targetError}</p> : null}
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

afterEach(cleanup);

describe("AssetCanvasEditorMode tileset wiring", () => {
  it("clears the selected tiles after an edit is submitted", async () => {
    mocks.canvas.send.mockClear();
    mocks.editor.onSubmit.mockResolvedValueOnce(true);
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
              prompt: "Add moss",
              tileset: { gridSize: 4, items: mocks.editor.sourceItems },
            },
          }}
          onBack={vi.fn()}
        />,
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "submit edit" }));

    await waitFor(() =>
      expect(mocks.canvas.send).toHaveBeenCalledWith({
        type: "selection.cleared",
      }),
    );
  });

  it("routes tree, canvas, review, and inspector events", () => {
    mocks.editor.review = {
      items: [],
      isResolving: false,
    };
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
    fireEvent.click(
      screen.getAllByRole("button", { name: "submit edit" }).at(-1)!,
    );

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
    mocks.editor.review = undefined;
  });

  it("renders empty branches when the workspace is unavailable or mode is unknown", () => {
    mocks.editorEnabled = false;
    const { container, rerender } = render(
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
    expect(container.textContent).toBe("");

    rerender(
      withI18n(
        <AssetCanvasEditorMode
          data={{ record: { mode: "unknown" } } as never}
          onBack={vi.fn()}
        />,
      ),
    );
    expect(container.textContent).toBe("");
    mocks.editorEnabled = true;
  });

  it("renders target validation and ignores submit without a target", () => {
    const previousItems = mocks.editor.sourceItems;
    const previousGridSize = mocks.editor.gridSize;
    const previousSelected = mocks.canvas.selectedCellIndexes;
    const manyItems = Array.from({ length: 257 }, (_, index) => ({
      id: `item-${index}`,
      label: `Item ${index}`,
      tiles: [[index % 20, Math.floor(index / 20)] as [number, number]],
    }));
    mocks.editor.sourceItems = manyItems as never;
    mocks.editor.gridSize = 20;
    mocks.canvas.selectedCellIndexes = Array.from(
      { length: 257 },
      (_, index) => index,
    );

    const view = render(
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
              tileset: { gridSize: 20, items: manyItems },
            },
          }}
          onBack={vi.fn()}
        />,
      ),
    );
    expect(
      screen.getByText("Select no more than 256 generated tiles."),
    ).toBeTruthy();

    mocks.editor.sourceItems = previousItems;
    mocks.editor.gridSize = previousGridSize;
    mocks.canvas.selectedCellIndexes = [];
    mocks.editor.onSubmit.mockClear();
    view.rerender(
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
              tileset: { gridSize: 4, items: previousItems },
            },
          }}
          onBack={vi.fn()}
        />,
      ),
    );
    fireEvent.click(
      screen.getAllByRole("button", { name: "submit edit" }).at(-1)!,
    );
    expect(mocks.editor.onSubmit).not.toHaveBeenCalled();
    mocks.canvas.selectedCellIndexes = previousSelected;
  });
});
