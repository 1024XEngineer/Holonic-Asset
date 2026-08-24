// @vitest-environment happy-dom

import { renderToStaticMarkup } from "react-dom/server";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { TilesetItem } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { TilesetCanvas } from "./tileset-canvas";

const items: TilesetItem[] = [
  {
    id: "sofa",
    label: "Sofa",
    imageUrl: "/sofa.png",
    tiles: [
      [0, 0],
      [1, 1],
    ],
  },
  { id: "label-only", label: "Label only", tiles: [[2, 2]] },
  { id: "empty", label: "Empty", imageUrl: "/empty.png", tiles: [] },
  {
    id: "outside",
    label: "Outside",
    imageUrl: "/outside.png",
    tiles: [[4, 0]],
  },
  {
    id: "generated",
    label: "Generated",
    tileUrls: ["/generated-1.png", "/generated-2.png"],
    tiles: [
      [2, 0],
      [3, 0],
    ],
  },
];

afterEach(cleanup);

describe("TilesetCanvas", () => {
  it("renders valid item images and filters invalid selected cells", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <TilesetCanvas
          model={{
            gridSize: 4,
            items,
            selectedCellIndexes: [0, 5, -1, 1.5, 16],
          }}
          onEvent={vi.fn()}
        />,
      ),
    );

    expect(html).toContain('src="/sofa.png"');
    expect(html).toContain('src="/generated-1.png"');
    expect(html).toContain('src="/generated-2.png"');
    expect(html).not.toContain('src="/empty.png"');
    expect(html).not.toContain('src="/outside.png"');
    expect(html).toContain("2 tiles selected");
    expect(html).toContain("opacity-40");
  });

  it("renders an empty state for an invalid grid size", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <TilesetCanvas
          model={{ gridSize: 0, items: [], selectedCellIndexes: [] }}
          onEvent={vi.fn()}
        />,
      ),
    );

    expect(html).toContain("No tileset grid");
    expect(html).toContain("No tiles selected");
  });

  it("opens an item-level comparison and emits review decisions", () => {
    const onEvent = vi.fn();
    const currentItem = items[4]!;
    const candidateItem = {
      ...currentItem,
      tileUrls: ["/candidate-1.png", "/generated-2.png"],
    };
    render(
      withI18n(
        <TilesetCanvas
          model={{
            gridSize: 4,
            items: [...items.slice(0, 4), candidateItem],
            selectedCellIndexes: [],
            review: {
              items: [
                {
                  itemId: currentItem.id,
                  currentItem,
                  candidateItem,
                },
              ],
              isResolving: false,
            },
          }}
          onEvent={onEvent}
        />,
      ),
    );

    const reviewButton = screen.getByRole("button", {
      name: "Review Generated changes",
    }) as HTMLButtonElement;
    expect(reviewButton.style.gridColumn).toBe("3 / span 2");
    expect(reviewButton.className).toContain("bg-emerald-200/35");
    fireEvent.click(reviewButton);
    expect(screen.getByRole("dialog")).toBeTruthy();
    expect(screen.getByText("Current")).toBeTruthy();
    expect(screen.getByText("Generated")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Apply" }));
    expect(onEvent).toHaveBeenCalledWith({
      type: "generation-review.resolved",
      applied: true,
    });
  });

  it("routes grid selection and closes a review from its dialog", () => {
    const onEvent = vi.fn();
    const item = items[0]!;
    const view = render(
      withI18n(
        <TilesetCanvas
          model={{
            gridSize: 4,
            items: [item],
            selectedCellIndexes: [],
            review: {
              items: [
                {
                  itemId: item.id,
                  currentItem: item,
                  candidateItem: item,
                },
              ],
              isResolving: false,
            },
          }}
          onEvent={onEvent}
        />,
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "Tile 1" }));
    expect(onEvent).toHaveBeenCalledWith({
      type: "cell.selection.toggled",
      gridCellIndex: 0,
    });

    const reviewButton = screen.getByRole("button", {
      name: "Review Sofa changes",
    });
    expect(reviewButton).toBeTruthy();
    fireEvent.click(reviewButton);
    expect(screen.getByRole("dialog")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(screen.queryByRole("dialog")).toBeNull();

    view.rerender(
      withI18n(
        <TilesetCanvas
          model={{
            gridSize: 4,
            items: [item],
            selectedCellIndexes: [],
            review: {
              items: [
                {
                  itemId: item.id,
                  currentItem: item,
                  candidateItem: { ...item, tiles: [[99, 99]] },
                },
              ],
              isResolving: false,
            },
          }}
          onEvent={onEvent}
        />,
      ),
    );
    expect(
      screen.queryByRole("button", { name: "Review Sofa changes" }),
    ).toBeNull();
  });

  it("disables review decisions while resolving", () => {
    const currentItem = items[4]!;
    render(
      withI18n(
        <TilesetCanvas
          model={{
            gridSize: 4,
            items,
            selectedCellIndexes: [],
            review: {
              items: [
                {
                  itemId: currentItem.id,
                  currentItem,
                  candidateItem: currentItem,
                },
              ],
              isResolving: true,
            },
          }}
          onEvent={vi.fn()}
        />,
      ),
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Review Generated changes" }),
    );

    expect(
      (screen.getByRole("button", { name: "Apply" }) as HTMLButtonElement)
        .disabled,
    ).toBe(true);
    expect(
      (screen.getByRole("button", { name: "Cancel" }) as HTMLButtonElement)
        .disabled,
    ).toBe(true);
  });
});
