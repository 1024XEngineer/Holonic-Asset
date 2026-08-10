import { describe, expect, it } from "vitest";

import type { AssetContentByType } from "./asset-content";

describe("asset content contracts", () => {
  it("preserves top-level metadata for every asset type", () => {
    const content = {
      character: {
        directionCount: 4,
        prototype: [],
        metadata: { source: "generator" },
      },
      object: { prototype: [], metadata: { source: "generator" } },
      tileSet: { metadata: { source: "generator" } },
      audio: { metadata: { source: "generator" } },
      uiset: { metadata: { source: "generator" } },
      scenery: { metadata: { source: "generator" } },
    } satisfies AssetContentByType;

    expect(content.scenery.metadata).toEqual({ source: "generator" });
  });
});
