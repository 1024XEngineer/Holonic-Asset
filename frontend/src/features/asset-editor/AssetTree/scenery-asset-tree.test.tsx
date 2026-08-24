// @vitest-environment happy-dom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { SceneryAssetTree } from "./scenery-asset-tree";

afterEach(cleanup);

const layers = [
  {
    id: "sky",
    label: "Sky",
    detail: "Backdrop",
    imageUrl: "/sky.png",
    blendMode: "normal" as const,
  },
  {
    id: "trees",
    label: "Trees",
    detail: "Foreground",
    imageUrl: "/trees.png",
    blendMode: "multiply" as const,
  },
];

describe("SceneryAssetTree", () => {
  it("renders layers through the shared shell", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <SceneryAssetTree
          layers={layers}
          selectedLayerId="trees"
          visibleLayerIds={["sky"]}
          onSelect={vi.fn()}
          onToggleVisibility={vi.fn()}
        />,
      ),
    );

    expect(html).toContain("Scene layers");
    expect(html).toContain("Sky");
    expect(html).toContain("Trees");
    expect(html).toContain("Backdrop");
    expect(html).toContain("Hide Sky");
  });

  it("routes layer selection and visibility actions", () => {
    const onSelect = vi.fn();
    const onToggleVisibility = vi.fn();
    render(
      withI18n(
        <SceneryAssetTree
          layers={layers}
          selectedLayerId="trees"
          visibleLayerIds={["sky"]}
          onSelect={onSelect}
          onToggleVisibility={onToggleVisibility}
        />,
      ),
    );

    fireEvent.click(screen.getByText("Trees"));
    fireEvent.click(screen.getByRole("button", { name: "Show Trees" }));

    expect(onSelect).toHaveBeenCalledWith("trees");
    expect(onToggleVisibility).toHaveBeenCalledWith("trees");
  });

  it("renders the empty state", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <SceneryAssetTree
          layers={[]}
          selectedLayerId={null}
          visibleLayerIds={[]}
          onSelect={vi.fn()}
          onToggleVisibility={vi.fn()}
        />,
      ),
    );

    expect(html).toContain("No scenery layers");
  });
});
