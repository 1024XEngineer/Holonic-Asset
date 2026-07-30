import type { EditorTilesetItem } from "@/features/asset-editor/types";

export type TilesetCanvasModel = {
  gridSize: number;
  items: EditorTilesetItem[];
  selectedCellIndexes: number[];
};

export type TilesetCanvasEvent = {
  type: "cell.selection.toggled";
  cellIndex: number;
};
