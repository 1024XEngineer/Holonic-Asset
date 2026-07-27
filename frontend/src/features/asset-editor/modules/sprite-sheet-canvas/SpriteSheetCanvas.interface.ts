export type SpriteSheetCanvasModel = {
  gridSize: number;
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
