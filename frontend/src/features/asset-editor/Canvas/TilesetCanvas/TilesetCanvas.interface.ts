import type { TilesetItem } from "@/model";

export type TilesetItemReview = {
  itemId: string;
  currentItem: TilesetItem;
  candidateItem: TilesetItem;
};

export type TilesetCanvasModel = {
  gridSize: number;
  items: readonly TilesetItem[];
  selectedCellIndexes: readonly number[];
  review?: {
    items: readonly TilesetItemReview[];
    isResolving: boolean;
  };
};

export type TilesetCanvasEvent =
  | {
      type: "cell.selection.toggled";
      gridCellIndex: number;
    }
  | { type: "generation-review.resolved"; applied: boolean };

export type TilesetCanvasProps = {
  model: TilesetCanvasModel;
  onEvent: (event: TilesetCanvasEvent) => void;
};
