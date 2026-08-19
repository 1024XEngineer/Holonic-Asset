import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { AssetTree } from "./asset-tree";

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

describe("AssetTree", () => {
  it("renders scenery layers through the shared tree", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <AssetTree
          kind="scenery"
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
});
