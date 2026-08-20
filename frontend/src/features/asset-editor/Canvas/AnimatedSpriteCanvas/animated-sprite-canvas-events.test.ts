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

  it("replaces the selection with frames from a marquee", () => {
    const onEvent = vi.fn();
    const actions = createAnimatedSpriteCanvasActions(onEvent, {
      nodeIds: ["idle"],
      frames: [{ nodeId: "idle", index: 0 }],
    });

    actions.onSelectFrames([
      { nodeId: "walk", index: 1 },
      { nodeId: "run", index: 2 },
    ]);

    expect(onEvent).toHaveBeenCalledWith({
      type: "selection.changed",
      selection: {
        nodeIds: ["walk", "run"],
        frames: [
          { nodeId: "walk", index: 1 },
          { nodeId: "run", index: 2 },
        ],
      },
    });
  });

  it("toggles animation groups from a modifier marquee", () => {
    const onEvent = vi.fn();
    const actions = createAnimatedSpriteCanvasActions(onEvent, {
      nodeIds: ["idle", "walk"],
      frames: [{ nodeId: "idle", index: 0 }],
    });

    actions.onSelectNodes(["idle", "run"], true);

    expect(onEvent).toHaveBeenCalledWith({
      type: "selection.changed",
      selection: {
        nodeIds: ["walk", "run"],
        frames: [],
      },
    });
  });

  it("emits clear, position, and review events", () => {
    const onEvent = vi.fn();
    const actions = createAnimatedSpriteCanvasActions(onEvent);

    actions.onClearSelection();
    actions.onNodePositionChange("idle", { x: 12, y: 24 });
    actions.onReviewResolve(true);

    expect(onEvent.mock.calls).toEqual([
      [
        {
          type: "selection.changed",
          selection: { nodeIds: [], frames: [] },
        },
      ],
      [
        {
          type: "node-position.committed",
          nodeId: "idle",
          position: { x: 12, y: 24 },
        },
      ],
      [{ type: "generation-review.resolved", applied: true }],
    ]);
  });
});
