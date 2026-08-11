import type { SceneryLayer } from "@/model";

export type SceneryCanvasModel = {
  layers: readonly SceneryLayer[];
  selectedLayerIds: readonly string[];
  visibleLayerIds: readonly string[];
};

export type SceneryCanvasEvent = {
  type: "layer.selection.toggled";
  layerId: string;
};

export type SceneryCanvasProps = {
  model: SceneryCanvasModel;
  onEvent: (event: SceneryCanvasEvent) => void;
};
