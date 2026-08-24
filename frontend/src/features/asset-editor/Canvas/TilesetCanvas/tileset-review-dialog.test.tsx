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

  it("keeps the dialog actions disabled while resolving", () => {
    render(
      withI18n(
        <TilesetReviewDialog
          item={{
            itemId: "empty",
            currentItem: { id: "empty", label: "Empty", tiles: [] },
            candidateItem: { id: "empty", label: "Empty", tiles: [] },
          }}
          isResolving
          onClose={vi.fn()}
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
  });
});
