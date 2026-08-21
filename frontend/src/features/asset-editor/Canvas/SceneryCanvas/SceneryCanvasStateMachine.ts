import { useEffect, useMemo, useReducer } from "react";

import type { SceneryLayer } from "@/model";

import type { SceneryCanvasEvent } from "./SceneryCanvas.interface";

export type SceneryCanvasState = {
  layerIds: string[];
  selectedLayerIds: string[];
  visibleLayerIds: string[];
};

export type SceneryCanvasStateEvent =
  | SceneryCanvasEvent
  | { type: "layer.visibility.toggled"; layerId: string }
  | { type: "layers.synced"; layerIds: string[] };

type SceneryCanvasPublicEvent = SceneryCanvasEvent;

export function createInitialSceneryCanvasState(
  layers: readonly SceneryLayer[],
): SceneryCanvasState {
  const layerIds = getUniqueLayerIds(layers);

  return {
    layerIds,
    selectedLayerIds: [],
    visibleLayerIds: layerIds.filter(
      (layerId) =>
        layers.find((layer) => layer.id === layerId)?.visible !== false,
    ),
  };
}

export function reduceSceneryCanvas(
  state: SceneryCanvasState,
  event: SceneryCanvasStateEvent,
): SceneryCanvasState {
  switch (event.type) {
    case "layers.synced":
      return reconcileLayers(state, unique(event.layerIds));
    case "layer.selection.toggled":
      if (!state.layerIds.includes(event.layerId)) return state;
      return {
        ...state,
        selectedLayerIds: toggle(state.selectedLayerIds, event.layerId),
      };
    case "layer.visibility.toggled":
      if (!state.layerIds.includes(event.layerId)) return state;
      return {
        ...state,
        visibleLayerIds: toggle(state.visibleLayerIds, event.layerId),
      };
  }
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
    selectedLayerIds: state.selectedLayerIds,
    visibleLayerIds: state.visibleLayerIds,
    send: (event: SceneryCanvasPublicEvent) => dispatch(event),
  };
}

function reconcileLayers(
  state: SceneryCanvasState,
  layerIds: string[],
): SceneryCanvasState {
  const currentLayerIds = new Set(state.layerIds);
  const selectedLayerIds = layerIds.filter((id) =>
    state.selectedLayerIds.includes(id),
  );
  const visibleLayerIds = layerIds.filter(
    (id) => !currentLayerIds.has(id) || state.visibleLayerIds.includes(id),
  );

  if (
    sameValues(state.layerIds, layerIds) &&
    sameValues(state.selectedLayerIds, selectedLayerIds) &&
    sameValues(state.visibleLayerIds, visibleLayerIds)
  ) {
    return state;
  }

  return { layerIds, selectedLayerIds, visibleLayerIds };
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
