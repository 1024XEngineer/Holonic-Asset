// @vitest-environment happy-dom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { TilesetAssetTree } from "./tileset-asset-tree";

afterEach(cleanup);

const items = [
  {
    id: "sofa",
    label: "Sofa",
    tiles: [
      [1, 2],
      [2, 2],
      [1, 3],
    ] as [number, number][],
  },
  {
    id: "lamp",
    label: "Lamp",
    tiles: [[5, 7]] as [number, number][],
  },
];

describe("TilesetAssetTree", () => {
  it("renders items through the shared shell", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <TilesetAssetTree
          items={items}
          selectedItemIds={[]}
          isTileSelected={() => false}
          onToggleItem={vi.fn()}
          onToggleTile={vi.fn()}
        />,
      ),
    );

    expect(html).toContain("Asset tree");
    expect(html).toContain("Sofa");
    expect(html).toContain("Lamp");
  });

  it("renders a relative grid with light selection states", () => {
    render(
      withI18n(
        <TilesetAssetTree
          items={items}
          selectedItemIds={[]}
          isTileSelected={(itemId, tileIndex) =>
            itemId === "sofa" && tileIndex === 0
          }
          onToggleItem={vi.fn()}
          onToggleTile={vi.fn()}
        />,
      ),
    );
    fireEvent.click(screen.getByRole("button", { name: "Expand Sofa" }));

    const selectedTile = screen.getByRole("button", {
      name: "Sofa tile at column 1, row 1",
    });
    const unselectedTile = screen.getByRole("button", {
      name: "Sofa tile at column 2, row 1",
    });
    expect(selectedTile.className).toContain("bg-muted");
    expect(selectedTile.className).not.toContain("bg-primary");
    expect(unselectedTile.className).toContain("bg-background");
    expect(unselectedTile.className).not.toContain("bg-foreground");
    expect(
      (
        screen.getByRole("button", {
          name: "Sofa tile at column 2, row 2",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
  });

  it("routes item and relative tile selection", () => {
    const onToggleItem = vi.fn();
    const onToggleTile = vi.fn();
    render(
      withI18n(
        <TilesetAssetTree
          items={items}
          selectedItemIds={[]}
          isTileSelected={() => false}
          onToggleItem={onToggleItem}
          onToggleTile={onToggleTile}
        />,
      ),
    );

    fireEvent.click(screen.getByText("Sofa"));
    fireEvent.click(screen.getByRole("button", { name: "Expand Lamp" }));
    fireEvent.click(
      screen.getByRole("button", {
        name: "Lamp tile at column 1, row 1",
      }),
    );

    expect(onToggleItem).toHaveBeenCalledWith("sofa");
    expect(onToggleTile).toHaveBeenCalledWith("lamp", 0);
  });

  it("renders the empty state", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <TilesetAssetTree
          items={[]}
          selectedItemIds={[]}
          isTileSelected={() => false}
          onToggleItem={vi.fn()}
          onToggleTile={vi.fn()}
        />,
      ),
    );

    expect(html).toContain("No tileset items");
  });

  it("does not render a grid for an item without tiles", () => {
    render(
      withI18n(
        <TilesetAssetTree
          items={[{ id: "empty", label: "Empty", tiles: [] }]}
          selectedItemIds={[]}
          isTileSelected={() => false}
          onToggleItem={vi.fn()}
          onToggleTile={vi.fn()}
        />,
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "Expand Empty" }));

    expect(screen.getByText("Empty")).toBeTruthy();
    expect(screen.queryByRole("grid")).toBeNull();
  });
});
