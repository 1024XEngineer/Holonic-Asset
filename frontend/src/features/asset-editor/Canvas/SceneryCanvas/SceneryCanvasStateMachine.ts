import { useEffect, useMemo, useReducer } from "react";

import type { SceneryLayer } from "@/model";

import type { SceneryCanvasEvent } from "./SceneryCanvas.interface";

export type SceneryCanvasState = {
  layerIds: string[];
  selectedLayers: string[];
  visibleLayers: string[];
};

export type SceneryCanvasStateEvent =
  | SceneryCanvasEvent
  | { type: "layer.visibility.toggled"; layerId: string }
  | { type: "layers.synced"; layerIds: string[] };

export function createInitialSceneryCanvasState(
  layers: readonly SceneryLayer[],
): SceneryCanvasState {
  const layerIds = getUniqueLayerIds(layers);

  return {
    layerIds,
    selectedLayers: [],
    visibleLayers: [...layerIds],
  };
}

export function reduceSceneryCanvas(
  state: SceneryCanvasState,
  event: SceneryCanvasStateEvent,
): SceneryCanvasState {
  if (event.type === "layers.synced") {
    return reconcileLayers(state, unique(event.layerIds));
  }

  if (!state.layerIds.includes(event.layerId)) return state;

  if (event.type === "layer.selection.toggled") {
    return {
      ...state,
      selectedLayers: toggle(state.selectedLayers, event.layerId),
    };
  }

  return {
    ...state,
    visibleLayers: toggle(state.visibleLayers, event.layerId),
  };
}

export function useSceneryCanvasStateMachine(layers: readonly SceneryLayer[]) {
  const layerIds = useMemo(() => getUniqueLayerIds(layers), [layers]);
  const [state, dispatch] = useReducer(
    reduceSceneryCanvas,
    layers,
    createInitialSceneryCanvasState,
  );

  useEffect(() => {
    dispatch({ type: "layers.synced", layerIds });
  }, [layerIds]);

  return {
    selectedLayers: state.selectedLayers,
    visibleLayers: state.visibleLayers,
    send: dispatch,
  };
}

function reconcileLayers(
  state: SceneryCanvasState,
  layerIds: string[],
): SceneryCanvasState {
  const currentLayerIds = new Set(state.layerIds);
  const selectedLayers = layerIds.filter((id) =>
    state.selectedLayers.includes(id),
  );
  const visibleLayers = layerIds.filter(
    (id) => !currentLayerIds.has(id) || state.visibleLayers.includes(id),
  );

  if (
    sameValues(state.layerIds, layerIds) &&
    sameValues(state.selectedLayers, selectedLayers) &&
    sameValues(state.visibleLayers, visibleLayers)
  ) {
    return state;
  }

  return { layerIds, selectedLayers, visibleLayers };
}

function getUniqueLayerIds(layers: readonly SceneryLayer[]) {
  return unique(layers.map((layer) => layer.id));
}

function unique(values: readonly string[]) {
  return [...new Set(values)];
}

function toggle(values: readonly string[], value: string) {
  return values.includes(value)
    ? values.filter((item) => item !== value)
    : [...values, value];
}

function sameValues(left: readonly string[], right: readonly string[]) {
  return (
    left.length === right.length &&
    left.every((value, index) => value === right[index])
  );
}
