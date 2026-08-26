// @vitest-environment happy-dom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { TilesetReviewDialog } from "./tileset-review-dialog";

afterEach(cleanup);

const currentItem = {
  id: "ground",
  label: "Ground",
  tiles: [
    [0, 0],
    [1, 0],
  ] as [number, number][],
  tileUrls: ["/ground-old.png", undefined],
};

describe("TilesetReviewDialog", () => {
  it("renders both tile-image and item-image comparisons", () => {
    render(
      withI18n(
        <TilesetReviewDialog
          item={{
            kind: "comparison",
            itemId: "ground",
            currentItem,
            candidateItem: {
              id: "ground",
              label: "Ground",
              tiles: currentItem.tiles,
              imageUrl: "/ground-generated.png",
            },
          }}
          isResolving={false}
          onClose={vi.fn()}
          onResolve={vi.fn()}
        />,
      ),
    );

    expect(screen.getByRole("dialog")).toBeTruthy();
    expect(document.querySelectorAll("img")).toHaveLength(2);
  });

  it("resolves cancel and apply actions", () => {
    const onResolve = vi.fn();
    const item = {
      kind: "comparison" as const,
      itemId: "empty",
      currentItem: { id: "empty", label: "Empty", tiles: [] },
      candidateItem: { id: "empty", label: "Empty", tiles: [] },
    };

    const view = render(
      withI18n(
        <TilesetReviewDialog
          item={item}
          isResolving={false}
          onClose={vi.fn()}
          onResolve={onResolve}
        />,
      ),
    );
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(screen.getByRole("button", { name: "Apply" }));

    expect(onResolve).toHaveBeenNthCalledWith(1, false);
    expect(onResolve).toHaveBeenNthCalledWith(2, true);

    view.unmount();
  });

  it("covers tall and image-less item previews", () => {
    render(
      withI18n(
        <TilesetReviewDialog
          item={{
            kind: "comparison",
            itemId: "wall",
            currentItem: {
              id: "wall",
              label: "Wall",
              tiles: [
                [0, 0],
                [0, 1],
              ],
            },
            candidateItem: {
              id: "wall",
              label: "Wall",
              tiles: [
                [0, 0],
                [0, 1],
              ],
            },
          }}
          isResolving={false}
          onClose={vi.fn()}
          onResolve={vi.fn()}
        />,
      ),
    );

    expect(screen.getAllByRole("button", { name: "Tile 1" })).toHaveLength(2);
    expect(document.querySelectorAll("img")).toHaveLength(0);
  });

  it("keeps the dialog actions disabled while resolving", () => {
    const onClose = vi.fn();
    render(
      withI18n(
        <TilesetReviewDialog
          item={{
            kind: "comparison",
            itemId: "empty",
            currentItem: { id: "empty", label: "Empty", tiles: [] },
            candidateItem: { id: "empty", label: "Empty", tiles: [] },
          }}
          isResolving
          onClose={onClose}
          onResolve={vi.fn()}
        />,
      ),
    );

    expect(
      (screen.getByRole("button", { name: "Cancel" }) as HTMLButtonElement)
        .disabled,
    ).toBe(true);
    expect(
      (screen.getByRole("button", { name: "Apply" }) as HTMLButtonElement)
        .disabled,
    ).toBe(true);
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).not.toHaveBeenCalled();
  });

  it("renders a new-item review without a current comparison", () => {
    render(
      withI18n(
        <TilesetReviewDialog
          item={{
            kind: "new-item",
            itemId: "candidate:0:Tree",
            candidateItem: {
              id: "candidate:0:Tree",
              label: "Tree",
              tiles: [[0, 0]],
              imageUrl: "/tree.png",
            },
          }}
          isResolving={false}
          onClose={vi.fn()}
          onResolve={vi.fn()}
        />,
      ),
    );

    expect(screen.getByText("Generated")).toBeTruthy();
    expect(screen.queryByText("Current")).toBeNull();
    expect(
      screen.getByText(
        "Review the generated item before adding it to this tileset.",
      ),
    ).toBeTruthy();
  });
});
