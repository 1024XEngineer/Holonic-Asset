import type { EditorSceneryLayer } from "@/features/asset-editor/types";

import type { SceneryCanvasEvent } from "./SceneryCanvas.interface";

export type SceneryCanvasState = {
  selectedLayers: string[];
  visibleLayers: string[];
};

export type SceneryCanvasStateEvent =
  | SceneryCanvasEvent
  | { type: "layer.visibility.toggled"; layerId: string };

function toggle<T>(values: T[], value: T) {
  return values.includes(value)
    ? values.filter((item) => item !== value)
    : [...values, value];
}

export function reduceSceneryCanvas(
  state: SceneryCanvasState,
  event: SceneryCanvasStateEvent,
): SceneryCanvasState {
  switch (event.type) {
    case "layer.selection.toggled":
      return {
        ...state,
        selectedLayers: toggle(state.selectedLayers, event.layerId),
      };
    case "layer.visibility.toggled":
      return {
        ...state,
        visibleLayers: toggle(state.visibleLayers, event.layerId),
      };
  }
}

export function createSceneryCanvasState(
  layers: EditorSceneryLayer[],
): SceneryCanvasState {
  return {
    selectedLayers: [],
    visibleLayers: layers.map((layer) => layer.id),
  };
}
