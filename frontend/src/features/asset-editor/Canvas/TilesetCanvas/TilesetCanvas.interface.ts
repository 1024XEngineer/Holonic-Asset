import type { TilesetItem } from "@/model";

export type TilesetCanvasModel = {
  gridSize: number;
  items: readonly TilesetItem[];
  selectedCellIndexes: readonly number[];
};

export type TilesetCanvasEvent = {
  type: "cell.selection.toggled";
  gridCellIndex: number;
};

export type TilesetCanvasProps = {
  model: TilesetCanvasModel;
  onEvent: (event: TilesetCanvasEvent) => void;
};
