import type { EditorTilesetItem } from "@/model";

export type TilesetCanvasModel = {
  gridSize: number;
  items: EditorTilesetItem[];
  selectedCellIndexes: number[];
};

export type TilesetCanvasEvent = {
  type: "cell.selection.toggled";
  cellIndex: number;
};

export type TilesetCanvasProps = {
  model: TilesetCanvasModel;
  onEvent: (event: TilesetCanvasEvent) => void;
};
