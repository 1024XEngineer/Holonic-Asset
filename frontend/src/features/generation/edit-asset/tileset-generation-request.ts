import type { CreateGenerationRequest, ItemTile } from "@/model";

import type { CreateTilesetItemRequest } from "../types";

export type TilesetGenerationTarget =
  | {
      kind: "item";
      position: ItemTile;
    }
  | {
      kind: "tiles";
      positions: ItemTile[];
    };

export function buildTilesetGenerationRequest({
  assetId,
  prompt,
  creatingReference,
  target,
}: {
  assetId: number;
  prompt: string;
  creatingReference?: { objectKey: string };
  target: TilesetGenerationTarget;
}): CreateGenerationRequest {
  const referenceParameters = creatingReference
    ? { creating_reference: creatingReference.objectKey }
    : {};

  if (target.kind === "item") {
    return {
      assetId,
      kind: "edit_tileset_item",
      creative_brief: prompt,
      parameters: {
        target: { position: toPosition(target.position) },
        ...referenceParameters,
      },
    };
  }

  return {
    assetId,
    kind: "edit_tiles",
    creative_brief: prompt,
    parameters: {
      targets: target.positions.map((position) => ({
        position: toPosition(position),
      })),
      ...referenceParameters,
    },
  };
}

export function buildAddTilesetItemGenerationRequest({
  assetId,
  request,
}: {
  assetId: number;
  request: CreateTilesetItemRequest;
}): CreateGenerationRequest {
  return {
    assetId,
    kind: "add_tileset_item",
    creative_brief: request.creativeBrief,
    parameters: {
      item: {
        name: request.itemName.trim(),
        description: request.itemDescription.trim(),
        shape: request.shape,
      },
    },
  };
}

function toPosition([x, y]: ItemTile) {
  return { x, y };
}
