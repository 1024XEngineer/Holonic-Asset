import { describe, expect, it, vi } from "vitest";

import { createAnimatedSpriteCanvasActions } from "../animated-sprite-canvas-events";
import type { AnimatedSpriteCanvasModel } from "../AnimatedSpriteCanvas.interface";
import type { AnimatedSpriteCanvasActions } from "./AnimatedSpriteCanvas.types";
import { AnimatedSpriteCanvasRuntime } from "./AnimatedSpriteCanvasRuntime";

const model = (): AnimatedSpriteCanvasModel => ({
  prototype: {
    format: "png-sprite-sheet",
    imageUrl: "prototype.png",
    frameWidth: 32,
    frameHeight: 32,
    columns: 4,
    rows: 1,
  },
  animations: [],
  selection: { nodeIds: [], frames: [] },
});

describe("AnimatedSpriteCanvasRuntime", () => {
  it("does not render when only the props wrapper changes", () => {
    const initialModel = model();
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: initialModel,
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });
    const render = vi.spyOn(
      runtime as unknown as { render: () => void },
      "render",
    );

    runtime.syncProps({
      model: { ...initialModel },
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });

    expect(render).not.toHaveBeenCalled();
  });

  it("renders when a scene input changes", () => {
    const initialModel = model();
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: initialModel,
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });
    const render = vi.spyOn(
      runtime as unknown as { render: () => void },
      "render",
    );

    runtime.syncProps({
      model: {
        ...initialModel,
        selection: { nodeIds: ["prototype"], frames: [] },
      },
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });

    expect(render).toHaveBeenCalledOnce();
  });

  it("forwards interactions to the latest actions without rendering", () => {
    const initialModel = model();
    const initialOnEvent = vi.fn();
    const latestOnEvent = vi.fn();
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: initialModel,
      actions: createAnimatedSpriteCanvasActions(initialOnEvent),
    });

    runtime.syncProps({
      model: { ...initialModel },
      actions: createAnimatedSpriteCanvasActions(latestOnEvent),
    });
    (
      runtime as unknown as { actions: AnimatedSpriteCanvasActions }
    ).actions.onSelect("prototype");

    expect(initialOnEvent).not.toHaveBeenCalled();
    expect(latestOnEvent).toHaveBeenCalledWith({
      type: "selection.changed",
      selection: { nodeIds: ["prototype"], frames: [] },
    });
  });
});
