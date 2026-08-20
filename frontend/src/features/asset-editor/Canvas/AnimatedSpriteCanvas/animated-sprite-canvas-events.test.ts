import { describe, expect, it, vi } from "vitest";

import { createAnimatedSpriteCanvasActions } from "./animated-sprite-canvas-events";

describe("createAnimatedSpriteCanvasActions", () => {
  it("keeps the existing selection when a frame is modifier-clicked", () => {
    const onEvent = vi.fn();
    const actions = createAnimatedSpriteCanvasActions(onEvent, {
      nodeIds: ["idle"],
      frames: [{ nodeId: "idle", index: 0 }],
    });

    actions.onSelectFrame("walk", 1, true);

    expect(onEvent).toHaveBeenCalledWith({
      type: "selection.changed",
      selection: {
        nodeIds: ["idle", "walk"],
        frames: [
          { nodeId: "idle", index: 0 },
          { nodeId: "walk", index: 1 },
        ],
      },
    });
  });

  it("keeps selected frames when an animation is modifier-clicked", () => {
    const onEvent = vi.fn();
    const actions = createAnimatedSpriteCanvasActions(onEvent, {
      nodeIds: ["idle"],
      frames: [{ nodeId: "idle", index: 0 }],
    });

    actions.onSelect("walk", true);

    expect(onEvent).toHaveBeenCalledWith({
      type: "selection.changed",
      selection: {
        nodeIds: ["idle", "walk"],
        frames: [{ nodeId: "idle", index: 0 }],
      },
    });
  });

  it("removes an already selected frame when it is modifier-clicked", () => {
    const onEvent = vi.fn();
    const actions = createAnimatedSpriteCanvasActions(onEvent, {
      nodeIds: ["idle", "walk"],
      frames: [
        { nodeId: "idle", index: 0 },
        { nodeId: "walk", index: 1 },
      ],
    });

    actions.onSelectFrame("idle", 0, true);

    expect(onEvent).toHaveBeenCalledWith({
      type: "selection.changed",
      selection: {
        nodeIds: ["walk"],
        frames: [{ nodeId: "walk", index: 1 }],
      },
    });
  });

  it("keeps an animation selected while it still has selected frames", () => {
    const onEvent = vi.fn();
    const actions = createAnimatedSpriteCanvasActions(onEvent, {
      nodeIds: ["idle"],
      frames: [
        { nodeId: "idle", index: 0 },
        { nodeId: "idle", index: 1 },
      ],
    });

    actions.onSelectFrame("idle", 0, true);

    expect(onEvent).toHaveBeenCalledWith({
      type: "selection.changed",
      selection: {
        nodeIds: ["idle"],
        frames: [{ nodeId: "idle", index: 1 }],
      },
    });
  });

  it("removes an already selected animation and its frames when modifier-clicked", () => {
    const onEvent = vi.fn();
    const actions = createAnimatedSpriteCanvasActions(onEvent, {
      nodeIds: ["idle", "walk"],
      frames: [
        { nodeId: "idle", index: 0 },
        { nodeId: "walk", index: 1 },
      ],
    });

    actions.onSelect("idle", true);

    expect(onEvent).toHaveBeenCalledWith({
      type: "selection.changed",
      selection: {
        nodeIds: ["walk"],
        frames: [{ nodeId: "walk", index: 1 }],
      },
    });
  });
});
