import type { SceneryCanvasDimensions, SceneryLayer } from "@/model";

export type SceneryCanvasModel = {
  layers: readonly SceneryLayer[];
  dimensions?: SceneryCanvasDimensions;
  selectedLayerIds: readonly string[];
  visibleLayerIds: readonly string[];
};

export type SceneryCanvasEvent =
  | { type: "layer.selection.toggled"; layerId: string }
  | { type: "layer.visibility.toggled"; layerId: string };

export type SceneryCanvasProps = {
  model: SceneryCanvasModel;
  onEvent: (event: SceneryCanvasEvent) => void;
};
