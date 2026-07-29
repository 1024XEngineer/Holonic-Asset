import type { EditorSceneryLayer } from "@/model";

export type SceneryCanvasModel = {
  layers: EditorSceneryLayer[];
  selectedLayerIds: string[];
  visibleLayerIds: string[];
};

export type SceneryCanvasEvent = {
  type: "layer.selection.toggled";
  layerId: string;
};
