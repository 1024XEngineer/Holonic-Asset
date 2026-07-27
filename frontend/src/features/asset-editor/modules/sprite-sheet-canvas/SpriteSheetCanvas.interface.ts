import type { EditorSpriteSheetItem } from "@/features/assets/domain";

export type SpriteSheetCanvasModel = {
  gridSize: number;
  items: EditorSpriteSheetItem[];
  selectedCellIndexes: number[];
};

export type SpriteSheetCanvasEvent = {
  type: "cell.selection.toggled";
  cellIndex: number;
};

export type SpriteSheetCanvasProps = {
  model: SpriteSheetCanvasModel;
  onEvent: (event: SpriteSheetCanvasEvent) => void;
};
