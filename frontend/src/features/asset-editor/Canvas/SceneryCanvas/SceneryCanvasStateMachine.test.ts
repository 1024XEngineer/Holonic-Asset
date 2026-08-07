import { describe, expect, it } from "vitest";

import type { SceneryLayer } from "@/model";

import {
  createInitialSceneryCanvasState,
  reduceSceneryCanvas,
} from "./SceneryCanvasStateMachine";

const layers: SceneryLayer[] = [
  {
    id: "sky",
    label: "Sky",
    detail: "Background",
    imageUrl: "/sky.png",
    blendMode: "normal",
  },
  {
    id: "trees",
    label: "Trees",
    detail: "Foreground",
    imageUrl: "/trees.png",
    blendMode: "multiply",
  },
];

describe("scenery canvas state", () => {
  it("starts with every layer visible and toggles selection independently", () => {
    let state = createInitialSceneryCanvasState(layers);

    state = reduceSceneryCanvas(state, {
      type: "layer.selection.toggled",
      layerId: "trees",
    });
    state = reduceSceneryCanvas(state, {
      type: "layer.visibility.toggled",
      layerId: "sky",
    });

    expect(state.selectedLayers).toEqual(["trees"]);
    expect(state.visibleLayers).toEqual(["trees"]);
  });

  it("preserves existing visibility while reconciling changed layers", () => {
    let state = createInitialSceneryCanvasState(layers);
    state = reduceSceneryCanvas(state, {
      type: "layer.visibility.toggled",
      layerId: "sky",
    });
    state = reduceSceneryCanvas(state, {
      type: "layer.selection.toggled",
      layerId: "trees",
    });

    state = reduceSceneryCanvas(state, {
      type: "layers.synced",
      layerIds: ["sky", "mist"],
    });

    expect(state).toEqual({
      layerIds: ["sky", "mist"],
      selectedLayers: [],
      visibleLayers: ["mist"],
    });
  });

  it("ignores events for layers outside the current record", () => {
    const state = createInitialSceneryCanvasState(layers);

    expect(
      reduceSceneryCanvas(state, {
        type: "layer.selection.toggled",
        layerId: "missing",
      }),
    ).toBe(state);
  });
});
