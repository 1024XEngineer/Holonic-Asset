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

  it("centers compact and wide stages and renders through the viewport", () => {
    const onZoomChange = vi.fn();
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: model(),
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
      onZoomChange,
    });
    const viewport = {
      scale: { x: 0.64 },
      setZoom: vi.fn(),
      moveCenter: vi.fn(),
    };
    const renderer = { render: vi.fn(), syncViewport: vi.fn() };
    const internals = runtime as unknown as {
      runtime: { app: { screen: { width: number; height: number } } };
      viewport: typeof viewport;
      renderer: typeof renderer;
      centerWorld: () => void;
      render: () => void;
      syncViewportGrid: () => void;
    };
    internals.viewport = viewport;
    internals.renderer = renderer;
    Object.defineProperty(internals.runtime.app, "screen", {
      value: { width: 500, height: 300 },
      configurable: true,
    });

    internals.centerWorld();
    internals.render();
    internals.syncViewportGrid();

    expect(viewport.moveCenter).toHaveBeenCalledWith(300, 300);
    expect(renderer.render).toHaveBeenCalledOnce();
    expect(renderer.syncViewport).toHaveBeenCalledTimes(2);
    expect(onZoomChange).toHaveBeenLastCalledWith(0.64);

    Object.defineProperty(internals.runtime.app, "screen", {
      value: { width: 800, height: 600 },
      configurable: true,
    });
    internals.centerWorld();
    expect(viewport.moveCenter).toHaveBeenLastCalledWith(650, 700);
  });

  it("renders only when an active animation reaches its frame duration", () => {
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: model(),
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });
    const renderer = { render: vi.fn(), syncViewport: vi.fn() };
    const viewport = { scale: { x: 1 } };
    const internals = runtime as unknown as {
      scene: { togglePlaying: (node: string) => void };
      lastAnimationTick: number;
      renderer: typeof renderer;
      viewport: typeof viewport;
      updateAnimation: () => void;
    };
    internals.renderer = renderer;
    internals.viewport = viewport;
    internals.lastAnimationTick = 0;
    const now = vi.spyOn(performance, "now");

    now.mockReturnValue(200);
    internals.updateAnimation();
    expect(renderer.render).not.toHaveBeenCalled();

    internals.scene.togglePlaying("prototype");
    now.mockReturnValue(359);
    internals.updateAnimation();
    expect(renderer.render).not.toHaveBeenCalled();

    now.mockReturnValue(360);
    internals.updateAnimation();
    expect(renderer.render).toHaveBeenCalledOnce();

    now.mockReturnValue(519);
    internals.updateAnimation();
    expect(renderer.render).toHaveBeenCalledOnce();
  });

  it("stops initialization after destruction or an unavailable stage", async () => {
    vi.spyOn(Assets, "load").mockResolvedValue({
      source: { scaleMode: "linear" },
    } as never);
    const destroyedRuntime = new AnimatedSpriteCanvasRuntime({
      model: model(),
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });
    destroyedRuntime.destroy();

    await destroyedRuntime.initialize({} as HTMLElement);

    const unavailableRuntime = new AnimatedSpriteCanvasRuntime({
      model: model(),
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });
    const initialize = vi
      .spyOn(
        (
          unavailableRuntime as unknown as {
            runtime: { initialize: (host: HTMLElement) => Promise<boolean> };
          }
        ).runtime,
        "initialize",
      )
      .mockResolvedValue(false);
    const host = {} as HTMLElement;

    await unavailableRuntime.initialize(host);

    expect(initialize).toHaveBeenCalledWith(host);
  });

  it("releases every initialized canvas resource only once", () => {
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: model(),
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });
    const ticker = { remove: vi.fn() };
    const stageRuntime = {
      initialized: true,
      app: { ticker },
      destroy: vi.fn(),
    };
    const resizeObserver = { disconnect: vi.fn() };
    const interaction = { destroy: vi.fn() };
    const viewport = {
      off: vi.fn(),
      removeFromParent: vi.fn(),
      destroy: vi.fn(),
    };
    const renderer = { destroy: vi.fn() };
    const internals = runtime as unknown as {
      runtime: typeof stageRuntime;
      resizeObserver: typeof resizeObserver;
      interaction: typeof interaction;
      viewport: typeof viewport | undefined;
      renderer: typeof renderer | undefined;
    };
    Object.defineProperty(internals, "runtime", { value: stageRuntime });
    internals.resizeObserver = resizeObserver;
    internals.interaction = interaction;
    internals.viewport = viewport;
    internals.renderer = renderer;

    runtime.destroy();
    runtime.destroy();

    expect(ticker.remove).toHaveBeenCalledOnce();
    expect(resizeObserver.disconnect).toHaveBeenCalledOnce();
    expect(interaction.destroy).toHaveBeenCalledOnce();
    expect(viewport.off).toHaveBeenCalledTimes(2);
    expect(viewport.removeFromParent).toHaveBeenCalledOnce();
    expect(viewport.destroy).toHaveBeenCalledWith({ children: true });
    expect(renderer.destroy).toHaveBeenCalledOnce();
    expect(stageRuntime.destroy).toHaveBeenCalledOnce();
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

  it("advances playback from elapsed time without a global interval gate", () => {
    const animationModel: AnimatedSpriteCanvasModel = {
      ...model(),
      animations: [
        {
          kind: "clip",
          id: "walk",
          label: "Walk",
          frameCount: 2,
          frameDurations: [50, 50],
        },
      ],
    };
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: animationModel,
      actions: createAnimatedSpriteCanvasActions(vi.fn()),
    });
    const internals = runtime as unknown as {
      lastAnimationTick: number;
      scene: {
        togglePlaying: (node: string) => void;
        getSnapshot: () => { previewFrames: ReadonlyMap<string, number> };
      };
      updateAnimation: () => void;
      render: () => void;
    };
    internals.lastAnimationTick = 1_000;
    internals.scene.togglePlaying("walk");
    const render = vi.spyOn(internals, "render");
    vi.spyOn(performance, "now")
      .mockReturnValueOnce(1_049)
      .mockReturnValueOnce(1_050);

    internals.updateAnimation();
    expect(internals.scene.getSnapshot().previewFrames.get("walk")).toBe(0);
    expect(render).not.toHaveBeenCalled();

    internals.updateAnimation();
    expect(internals.scene.getSnapshot().previewFrames.get("walk")).toBe(1);
    expect(render).toHaveBeenCalledOnce();
  });

  it("forwards every interaction to the latest actions without rendering", () => {
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
    const actions = (
      runtime as unknown as { actions: AnimatedSpriteCanvasActions }
    ).actions;
    actions.onSelect("prototype");
    actions.onSelectFrame("prototype", 0, true);
    actions.onSelectFrames([{ nodeId: "prototype", index: 1 }], true);
    actions.onSelectNodes(["prototype"], true);
    actions.onClearSelection();
    actions.onNodePositionChange("prototype", { x: 12, y: 24 });
    actions.onReviewResolve(true);

    expect(initialOnEvent).not.toHaveBeenCalled();
    expect(latestOnEvent.mock.calls).toEqual([
      [
        {
          type: "selection.changed",
          selection: { nodeIds: ["prototype"], frames: [] },
        },
      ],
      [
        {
          type: "selection.changed",
          selection: {
            nodeIds: ["prototype"],
            frames: [{ nodeId: "prototype", index: 0 }],
          },
        },
      ],
      [
        {
          type: "selection.changed",
          selection: {
            nodeIds: ["prototype"],
            frames: [{ nodeId: "prototype", index: 1 }],
          },
        },
      ],
      [
        {
          type: "selection.changed",
          selection: { nodeIds: ["prototype"], frames: [] },
        },
      ],
      [
        {
          type: "selection.changed",
          selection: { nodeIds: [], frames: [] },
        },
      ],
      [
        {
          type: "node-position.committed",
          nodeId: "prototype",
          position: { x: 12, y: 24 },
        },
      ],
      [{ type: "generation-review.resolved", applied: true }],
    ]);
  });
});
