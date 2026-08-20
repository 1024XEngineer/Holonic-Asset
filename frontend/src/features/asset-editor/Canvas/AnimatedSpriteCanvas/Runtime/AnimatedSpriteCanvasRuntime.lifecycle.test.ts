// @vitest-environment happy-dom

import { Assets } from "pixi.js";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { AnimatedSpriteCanvasModel } from "../AnimatedSpriteCanvas.interface";
import type {
  AnimatedSpriteCanvasActions,
  AnimatedSpriteStageContext,
} from "./AnimatedSpriteCanvas.types";
import { AnimatedSpriteCanvasRuntime } from "./AnimatedSpriteCanvasRuntime";

const boundary = vi.hoisted(() => ({
  interactionContext: undefined as AnimatedSpriteStageContext | undefined,
  interactionDestroy: vi.fn(),
  rendererDestroy: vi.fn(),
  rendererRender: vi.fn(),
  rendererSyncViewport: vi.fn(),
  resizeCallback: undefined as (() => void) | undefined,
  resizeDisconnect: vi.fn(),
  resizeObserve: vi.fn(),
  viewport: undefined as
    | {
        scale: { x: number };
        resize: (width: number, height: number) => void;
      }
    | undefined,
  viewportAddChild: vi.fn(),
  viewportClampZoom: vi.fn(),
  viewportDestroy: vi.fn(),
  viewportDrag: vi.fn(),
  viewportMoveCenter: vi.fn(),
  viewportOff: vi.fn(),
  viewportOn: vi.fn(),
  viewportRemoveFromParent: vi.fn(),
  viewportResize: vi.fn(),
  viewportSetZoom: vi.fn(),
  viewportWheel: vi.fn(),
}));

vi.mock("pixi-viewport", () => ({
  Viewport: class {
    eventMode = "none";
    scale = { x: 1 };
    screenWidth: number;
    screenHeight: number;
    x = 0;
    y = 0;

    constructor(options: { screenWidth: number; screenHeight: number }) {
      this.screenWidth = options.screenWidth;
      this.screenHeight = options.screenHeight;
      boundary.viewport = this;
    }

    drag(options: unknown) {
      boundary.viewportDrag(options);
      return this;
    }
    wheel() {
      boundary.viewportWheel();
      return this;
    }
    clampZoom(options: unknown) {
      boundary.viewportClampZoom(options);
      return this;
    }
    addChild(child: unknown) {
      boundary.viewportAddChild(child);
    }
    on(event: string, callback: () => void) {
      boundary.viewportOn(event, callback);
    }
    off(event: string, callback: () => void) {
      boundary.viewportOff(event, callback);
    }
    resize(width: number, height: number) {
      this.screenWidth = width;
      this.screenHeight = height;
      boundary.viewportResize(width, height);
    }
    setZoom(scale: number) {
      this.scale.x = scale;
      boundary.viewportSetZoom(scale);
    }
    moveCenter(x: number, y: number) {
      boundary.viewportMoveCenter(x, y);
    }
    removeFromParent() {
      boundary.viewportRemoveFromParent();
    }
    destroy(options: unknown) {
      boundary.viewportDestroy(options);
    }
  },
}));

vi.mock("../Renderer/AnimatedSpriteStageRenderer", () => ({
  AnimatedSpriteStageRenderer: class {
    render(state: unknown, model: unknown) {
      boundary.rendererRender(state, model);
    }
    syncViewport(viewport: unknown, prototype: unknown) {
      boundary.rendererSyncViewport(viewport, prototype);
    }
    destroy() {
      boundary.rendererDestroy();
    }
  },
}));

vi.mock("../Interaction/AnimatedSpriteStageInteraction", () => ({
  AnimatedSpriteStageInteraction: class {
    constructor(
      _canvas: HTMLCanvasElement,
      context: AnimatedSpriteStageContext,
    ) {
      boundary.interactionContext = context;
    }
    destroy() {
      boundary.interactionDestroy();
    }
  },
}));

const model = (): AnimatedSpriteCanvasModel => ({
  prototype: {
    format: "png-sprite-sheet",
    imageUrl: "prototype.png",
    frameWidth: 32,
    frameHeight: 32,
    columns: 4,
    rows: 1,
  },
  animations: [{ id: "idle", kind: "clip", label: "Idle", frameCount: 4 }],
  selection: { nodeIds: [], frames: [] },
});

function actions(): AnimatedSpriteCanvasActions {
  return {
    onSelect: vi.fn(),
    onSelectFrame: vi.fn(),
    onSelectFrames: vi.fn(),
    onSelectNodes: vi.fn(),
    onClearSelection: vi.fn(),
    onNodePositionChange: vi.fn(),
    onReviewResolve: vi.fn(),
  };
}

describe("AnimatedSpriteCanvasRuntime lifecycle", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    boundary.interactionContext = undefined;
    boundary.resizeCallback = undefined;
    boundary.viewport = undefined;
    vi.stubGlobal(
      "ResizeObserver",
      class {
        constructor(callback: () => void) {
          boundary.resizeCallback = callback;
        }
        observe(target: Element) {
          boundary.resizeObserve(target);
        }
        disconnect() {
          boundary.resizeDisconnect();
        }
      },
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("initializes the stage boundary and exposes a working interaction context", async () => {
    vi.spyOn(Assets, "load").mockResolvedValue({
      source: { scaleMode: "linear" },
    } as never);
    const ticker = { add: vi.fn(), remove: vi.fn() };
    const canvas = document.createElement("canvas");
    const stage = { addChild: vi.fn() };
    const stageRuntime = {
      initialized: true,
      app: {
        screen: { width: 800, height: 600 },
        renderer: { events: {} },
        ticker,
        stage,
        canvas,
      },
      initialize: vi.fn().mockResolvedValue(true),
      destroy: vi.fn(),
    };
    const onZoomChange = vi.fn();
    const runtime = new AnimatedSpriteCanvasRuntime({
      model: model(),
      actions: actions(),
      onZoomChange,
    });
    Object.defineProperty(runtime, "runtime", { value: stageRuntime });
    const host = document.createElement("div");

    await runtime.initialize(host);
    await Promise.resolve();

    expect(stageRuntime.initialize).toHaveBeenCalledWith(host);
    expect(boundary.viewportDrag).toHaveBeenCalledWith({
      mouseButtons: "middle",
    });
    expect(boundary.viewportWheel).toHaveBeenCalledOnce();
    expect(boundary.viewportClampZoom).toHaveBeenCalledOnce();
    expect(stage.addChild).toHaveBeenCalledOnce();
    expect(boundary.resizeObserve).toHaveBeenCalledWith(host);
    expect(ticker.add).toHaveBeenCalledOnce();
    expect(boundary.viewportMoveCenter).toHaveBeenCalledWith(650, 700);
    expect(onZoomChange).toHaveBeenCalledWith(0.64);
    expect(boundary.rendererRender).toHaveBeenCalled();

    const context = boundary.interactionContext!;
    expect(context.getAnimations()).toHaveLength(1);
    expect(context.getPrototype().columns).toBe(4);
    expect(context.getReview()).toBeUndefined();
    expect(context.getDragStep()).toBe(3);
    context.moveNode("idle", { x: 40, y: 60 });
    expect(context.getScene().positions.idle).toEqual({ x: 40, y: 60 });
    context.setMarquee({ start: { x: 1, y: 2 }, end: { x: 3, y: 4 } });
    expect(context.getScene().marquee).toEqual({
      start: { x: 1, y: 2 },
      end: { x: 3, y: 4 },
    });
    context.toggleExpanded("idle");
    expect(context.getScene().expanded.has("idle")).toBe(true);
    context.togglePlaying("idle");
    context.render();

    boundary.resizeCallback!();
    expect(boundary.viewportResize).toHaveBeenCalledWith(800, 600);

    runtime.syncProps({
      model: { ...model(), selection: { nodeIds: ["idle"], frames: [] } },
      actions: actions(),
      onZoomChange,
    });
    await Promise.resolve();
    expect(Assets.load).toHaveBeenCalled();

    runtime.destroy();
    expect(ticker.remove).toHaveBeenCalledOnce();
    expect(boundary.resizeDisconnect).toHaveBeenCalledOnce();
    expect(boundary.interactionDestroy).toHaveBeenCalledOnce();
    expect(boundary.viewportOff).toHaveBeenCalledTimes(2);
    expect(boundary.rendererDestroy).toHaveBeenCalledOnce();
    expect(boundary.viewportDestroy).toHaveBeenCalledWith({ children: true });
    expect(stageRuntime.destroy).toHaveBeenCalledOnce();
  });
});
