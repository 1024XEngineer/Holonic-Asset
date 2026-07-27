import type { EditorSceneryLayer } from "../../domain";

export type SceneryCanvasModel = {
  layers: EditorSceneryLayer[];
  selectedLayerIds: string[];
  visibleLayerIds: string[];
};

export type SceneryCanvasEvent = {
  type: "layer.selection.toggled";
  layerId: string;
};

export type SceneryCanvasProps = {
  model: SceneryCanvasModel;
  onEvent: (event: SceneryCanvasEvent) => void;
};
