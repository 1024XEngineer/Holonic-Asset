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
  it("sets a finite zoom around the current viewport center", () => {
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: model(),
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });
    const setZoom = vi.fn();
    const syncViewportGrid = vi.spyOn(
      runtime as unknown as { syncViewportGrid: () => void },
      "syncViewportGrid",
    );
    (runtime as unknown as { viewport: { setZoom: typeof setZoom } }).viewport =
      { setZoom };

    runtime.setZoom(1.25);
    runtime.setZoom(Number.NaN);

    expect(setZoom).toHaveBeenCalledWith(1.25, true);
    expect(syncViewportGrid).toHaveBeenCalledOnce();
  });

  it("notifies the latest zoom callback when the viewport is available", () => {
    const onZoomChange = vi.fn();
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: model(),
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
      onZoomChange,
    });
    (runtime as unknown as { viewport: { scale: { x: number } } }).viewport = {
      scale: { x: 0.88 },
    };

    (runtime as unknown as { notifyZoomChange: () => void }).notifyZoomChange();

    expect(onZoomChange).toHaveBeenCalledWith(0.88);
  });

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
