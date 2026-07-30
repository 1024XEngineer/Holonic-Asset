import type { EditorSceneryLayer } from "@/features/asset-editor/types";

export type SceneryCanvasModel = {
  layers: EditorSceneryLayer[];
  selectedLayerIds: string[];
  visibleLayerIds: string[];
};

export type SceneryCanvasEvent = {
  type: "layer.selection.toggled";
  layerId: string;
};
