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

    expect(state.selectedLayerIds).toEqual(["trees"]);
    expect(state.visibleLayerIds).toEqual(["trees"]);
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
      selectedLayerIds: [],
      visibleLayerIds: ["mist"],
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

  it("deduplicates layers and returns the same state for a no-op sync", () => {
    const state = createInitialSceneryCanvasState([layers[0], layers[0]]);

    expect(state.layerIds).toEqual(["sky"]);
    expect(
      reduceSceneryCanvas(state, {
        type: "layers.synced",
        layerIds: ["sky", "sky"],
      }),
    ).toBe(state);
  });

  it("toggles selected and visible layers off and back on", () => {
    let state = createInitialSceneryCanvasState(layers);

    for (const type of [
      "layer.selection.toggled",
      "layer.selection.toggled",
      "layer.visibility.toggled",
      "layer.visibility.toggled",
    ] as const) {
      state = reduceSceneryCanvas(state, { type, layerId: "sky" });
    }

    expect(state.selectedLayerIds).toEqual([]);
    expect(state.visibleLayerIds).toEqual(["trees", "sky"]);
  });
});
