import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { AssetTree } from "./asset-tree";

describe("AssetTree", () => {
  it("renders shared tree chrome and content", () => {
    const html = renderToStaticMarkup(
      <AssetTree
        title="Asset tree"
        description="Mode-specific assets"
        count={2}
        footer={<button type="button">Action</button>}
      >
        <span>Node</span>
      </AssetTree>,
    );

    expect(html).toContain("Asset tree");
    expect(html).toContain("Mode-specific assets");
    expect(html).toContain("Node");
    expect(html).toContain("Action");
    expect(html).toContain(">2<");
  });

  it("renders the shared empty state instead of children", () => {
    const html = renderToStaticMarkup(
      <AssetTree
        title="Asset tree"
        description="Mode-specific assets"
        count={0}
        emptyMessage="Nothing here"
      >
        <span>Hidden node</span>
      </AssetTree>,
    );

    expect(html).toContain("Nothing here");
    expect(html).not.toContain("Hidden node");
  });
});
