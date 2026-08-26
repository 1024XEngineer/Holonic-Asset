import type { ItemTile } from "@/model/item-tile";

export type CreateTilesetItemRequest = {
  itemName: string;
  itemDescription: string;
  shape: ItemTile[];
  creativeBrief: string;
};
