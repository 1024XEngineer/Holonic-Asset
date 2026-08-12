import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

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
];

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
});
