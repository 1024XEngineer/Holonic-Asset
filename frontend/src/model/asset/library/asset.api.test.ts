import { describe, expect, it } from "vitest";

import { toAssetGroups } from "./asset.api";

describe("toAssetGroups", () => {
  it("maps API assets into typed groups with defaults and stable ordering", () => {
    expect(
      toAssetGroups([
        {
          assetId: 8,
          projectId: 1,
          name: "Barrel",
          description: "Wooden prop",
          dimensions: {},
          perspective: "Top-Down",
          type: "object",
          version: 2,
          tags: ["prop"],
        },
        {
          assetId: 9,
          projectId: 1,
          name: "Ground",
          description: "Tile set",
          dimensions: {},
          perspective: "Top-Down",
          type: "tileSet",
          version: 1,
          tags: [],
        },
        {
          assetId: 10,
          projectId: 1,
          name: "Crate",
          description: "Another prop",
          dimensions: {},
          perspective: "Top-Down",
          type: "object",
          version: 3,
          tags: null,
        },
      ]),
    ).toMatchObject([
      {
        kind: "object",
        assets: [
          { id: "8", version: "v2", tags: ["prop"] },
          { id: "10", version: "v3", tags: [] },
        ],
      },
      {
        kind: "tileset",
        assets: [{ id: "9", version: "v1", tags: [] }],
      },
    ]);
  });
});
