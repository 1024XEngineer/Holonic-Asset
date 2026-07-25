import { useReducer } from "react";

import type { EditorSceneryLayer } from "@/domain/asset";

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

function reducer(
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

export function useSceneryCanvasStateMachine(layers: EditorSceneryLayer[]) {
  const [state, dispatch] = useReducer(
    reducer,
    layers,
    (initialLayers): SceneryCanvasState => ({
      selectedLayers: [],
      visibleLayers: initialLayers.map((layer) => layer.id),
    }),
  );
  return {
    ...state,
    send: (event: SceneryCanvasStateEvent) => dispatch(event),
  };
}
