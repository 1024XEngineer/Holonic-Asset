import { Assets } from "pixi.js";
import { afterEach, describe, expect, it, vi } from "vitest";

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
  afterEach(() => vi.restoreAllMocks());

  it("preloads every independent prototype direction", async () => {
    const load = vi.spyOn(Assets, "load").mockResolvedValue({
      source: { scaleMode: "linear" },
    } as never);
    const directionalModel = {
      ...model(),
      prototype: {
        ...model().prototype,
        frameUrls: ["front.png", "back.png"],
      },
    };
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: directionalModel,
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });

    await (
      runtime as unknown as {
        preloadAnimatedSpriteTextures: (
          value: AnimatedSpriteCanvasModel,
        ) => Promise<void>;
      }
    ).preloadAnimatedSpriteTextures(directionalModel);

    expect(load).toHaveBeenCalledWith("prototype.png");
    expect(load).toHaveBeenCalledWith("front.png");
    expect(load).toHaveBeenCalledWith("back.png");
  });

  it("preloads every independent animation frame", async () => {
    const load = vi.spyOn(Assets, "load").mockResolvedValue({
      source: { scaleMode: "linear" },
    } as never);
    const animationModel = {
      ...model(),
      animations: [
        {
          id: "pending",
          kind: "clip" as const,
          label: "Pending",
          frameCount: 0,
        },
        {
          id: "idle",
          kind: "clip" as const,
          label: "Idle",
          frameCount: 1,
          spriteSheet: {
            ...model().prototype,
            imageUrl: "shared.png",
          },
        },
        {
          id: "walk",
          kind: "clip" as const,
          label: "Walk",
          frameCount: 2,
          spriteSheet: {
            ...model().prototype,
            imageUrl: "walk-1.png",
            frameUrls: ["walk-1.png", "walk-2.png"],
          },
        },
      ],
    };
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: animationModel,
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });

    await (
      runtime as unknown as {
        preloadAnimatedSpriteTextures: (
          value: AnimatedSpriteCanvasModel,
        ) => Promise<void>;
      }
    ).preloadAnimatedSpriteTextures(animationModel);

    expect(load).toHaveBeenCalledWith("walk-1.png");
    expect(load).toHaveBeenCalledWith("walk-2.png");
  });

  it("deduplicates texture URLs and isolates failed frame loads", async () => {
    const loadedTexture = { source: { scaleMode: "linear" } };
    const load = vi
      .spyOn(Assets, "load")
      .mockImplementation((url) =>
        typeof url === "string" && url === "broken.png"
          ? Promise.reject(new Error("frame unavailable"))
          : Promise.resolve(loadedTexture as never),
      );
    const animationModel = {
      ...model(),
      prototype: { ...model().prototype, imageUrl: "shared.png" },
      animations: [
        {
          id: "walk",
          kind: "clip" as const,
          label: "Walk",
          frameCount: 2,
          spriteSheet: {
            ...model().prototype,
            imageUrl: "shared.png",
            frameUrls: ["shared.png", "broken.png"],
          },
        },
      ],
    };
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: animationModel,
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });
    const internals = runtime as unknown as {
      preloadAnimatedSpriteTextures: (
        value: AnimatedSpriteCanvasModel,
      ) => Promise<void>;
      render: () => void;
      unavailableTextureUrls: Set<string>;
    };
    internals.unavailableTextureUrls.add("shared.png");
    const render = vi.spyOn(internals, "render");

    await internals.preloadAnimatedSpriteTextures(animationModel);

    expect(load).toHaveBeenCalledTimes(2);
    expect(load).toHaveBeenCalledWith("shared.png");
    expect(load).toHaveBeenCalledWith("broken.png");
    expect(loadedTexture.source.scaleMode).toBe("nearest");
    expect(internals.unavailableTextureUrls.has("shared.png")).toBe(false);
    expect(internals.unavailableTextureUrls.has("broken.png")).toBe(true);
    expect(render).toHaveBeenCalledOnce();
  });

  it("does not render after texture loading when the runtime is destroyed", async () => {
    vi.spyOn(Assets, "load").mockResolvedValue({
      source: { scaleMode: "linear" },
    } as never);
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: model(),
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });
    const internals = runtime as unknown as {
      destroyed: boolean;
      preloadAnimatedSpriteTextures: (
        value: AnimatedSpriteCanvasModel,
      ) => Promise<void>;
      render: () => void;
    };
    internals.destroyed = true;
    const render = vi.spyOn(internals, "render");

    await internals.preloadAnimatedSpriteTextures(model());

    expect(render).not.toHaveBeenCalled();
  });

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
