import { describe, expect, it } from "vitest";

import {
  buildAddTilesetItemGenerationRequest,
  buildTilesetGenerationRequest,
} from "./tileset-generation-request";

describe("buildTilesetGenerationRequest", () => {
  it("builds a new item request with its requested footprint", () => {
    expect(
      buildAddTilesetItemGenerationRequest({
        assetId: 9,
        request: {
          itemName: " Oak tree ",
          itemDescription: "A broad old oak",
          shape: [
            [0, 0],
            [1, 0],
          ],
          creativeBrief: "Warm pixel-art bark and dense leaves",
        },
      }),
    ).toEqual({
      assetId: 9,
      kind: "add_tileset_item",
      creative_brief: "Warm pixel-art bark and dense leaves",
      parameters: {
        item: {
          name: "Oak tree",
          description: "A broad old oak",
          shape: [
            [0, 0],
            [1, 0],
          ],
        },
      },
    });
  });

  it("builds an item edit with its anchor and reference", () => {
    expect(
      buildTilesetGenerationRequest({
        assetId: 9,
        prompt: "Add moss",
        creatingReference: { objectKey: "uploads/moss.png" },
        target: {
          kind: "item",
          position: [0, 0],
        },
      }),
    ).toEqual({
      assetId: 9,
      kind: "edit_tileset_item",
      creative_brief: "Add moss",
      parameters: {
        target: { position: { x: 0, y: 0 } },
        creating_reference: "uploads/moss.png",
      },
    });
  });

  it("builds a multi-tile edit with stable positions", () => {
    expect(
      buildTilesetGenerationRequest({
        assetId: 9,
        prompt: "Add moss",
        target: {
          kind: "tiles",
          positions: [
            [0, 0],
            [1, 0],
            [3, 3],
          ],
        },
      }),
    ).toEqual({
      assetId: 9,
      kind: "edit_tiles",
      creative_brief: "Add moss",
      parameters: {
        targets: [
          { position: { x: 0, y: 0 } },
          { position: { x: 1, y: 0 } },
          { position: { x: 3, y: 3 } },
        ],
      },
    });
  });
});
