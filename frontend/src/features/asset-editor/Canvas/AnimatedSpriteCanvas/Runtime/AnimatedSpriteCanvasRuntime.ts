import { Viewport } from "pixi-viewport";
import { Assets, Container } from "pixi.js";
import { AnimatedSpriteStageInteraction } from "../Interaction/AnimatedSpriteStageInteraction";
import { AnimatedSpriteStageRenderer } from "../Renderer/AnimatedSpriteStageRenderer";
import {
  getAnimatedSpriteMaxScale,
  getAnimatedSpritePixelScale,
} from "../animated-sprite-scale";
import {
  FRAME_SIZE,
  INITIAL_SCALE,
  MAX_SOURCE_PIXEL_SCREEN_SIZE,
  MIN_SCALE,
} from "./AnimatedSpriteStage.constants";
import type {
  AnimatedSpriteCanvasRuntimeProps,
  AnimatedSpriteStageContext,
} from "./AnimatedSpriteCanvas.types";
import { AnimatedSpriteScene } from "./AnimatedSpriteScene";
import { StageRuntime } from "./StageRuntime";

export class AnimatedSpriteCanvasRuntime {
  private readonly runtime = new StageRuntime();
  private interaction?: AnimatedSpriteStageInteraction;
  private resizeObserver?: ResizeObserver;
  private renderer?: AnimatedSpriteStageRenderer;
  private viewport?: Viewport;
  private props: AnimatedSpriteCanvasRuntimeProps;
  private lastAnimationFrame = performance.now();
  private readonly unavailableTextureUrls = new Set<string>();
  private readonly scene: AnimatedSpriteScene;
  private destroyed = false;

  constructor(props: AnimatedSpriteCanvasRuntimeProps) {
    this.props = props;
    this.scene = new AnimatedSpriteScene(props.model);
  }

  async initialize(host: HTMLElement) {
    await this.preloadAnimatedSpriteTextures(this.props.model);
    if (this.destroyed) return;
    await this.runtime.initialize(host);
    if (this.destroyed) return;
    const { app } = this.runtime;
    const viewport = new Viewport({
      screenWidth: app.screen.width,
      screenHeight: app.screen.height,
      events: app.renderer.events,
      ticker: app.ticker,
    });
    viewport.eventMode = "static";
    viewport
      .drag({ mouseButtons: "middle" })
      .wheel()
      .clampZoom({
        minScale: MIN_SCALE,
        maxScale: getAnimatedSpriteMaxScale(
          this.props.model.prototype,
          FRAME_SIZE,
          MAX_SOURCE_PIXEL_SCREEN_SIZE,
        ),
      });
    app.stage.addChild(viewport);
    this.viewport = viewport;
    const world = new Container();
    viewport.addChild(world);
    this.renderer = new AnimatedSpriteStageRenderer(app.stage, world);
    viewport.on("moved", this.syncViewportGrid);
    viewport.on("zoomed", this.syncViewportGrid);
    const context: AnimatedSpriteStageContext = {
      viewport,
      actions: this.props.actions,
      getAnimations: () => this.props.model.animations,
      getPrototype: () => this.props.model.prototype,
      getScene: () => this.scene.getSnapshot(),
      moveNode: (node, position) => this.scene.moveNode(node, position),
      setMarquee: (marquee) => this.scene.setMarquee(marquee),
      getDragStep: () =>
        getAnimatedSpritePixelScale(this.props.model.prototype, FRAME_SIZE),
      toggleExpanded: (node) => this.scene.toggleExpanded(node),
      togglePlaying: (node) => this.scene.togglePlaying(node),
      render: () => this.render(),
    };
    this.interaction = new AnimatedSpriteStageInteraction(app.canvas, context);
    this.resizeObserver = new ResizeObserver(() => {
      viewport.resize(app.screen.width, app.screen.height);
      this.render();
    });
    this.resizeObserver.observe(host);
    app.ticker.add(this.updateAnimation);
    this.centerWorld();
    this.syncProps(this.props);
  }

  syncProps(props: AnimatedSpriteCanvasRuntimeProps) {
    this.props = props;
    this.scene.synchronize(props.model);
    if (this.viewport && this.renderer)
      void this.preloadAnimatedSpriteTextures(props.model);
    this.render();
  }

  destroy() {
    if (this.destroyed) return;
    this.destroyed = true;
    this.runtime.app.ticker.remove(this.updateAnimation);
    this.resizeObserver?.disconnect();
    this.interaction?.destroy();
    this.viewport?.off("moved", this.syncViewportGrid);
    this.viewport?.off("zoomed", this.syncViewportGrid);
    this.viewport?.removeFromParent();
    this.viewport?.destroy({ children: true });
    this.viewport = undefined;
    this.runtime.destroy();
  }

  private async preloadAnimatedSpriteTextures(
    model: AnimatedSpriteCanvasRuntimeProps["model"],
  ) {
    const urls = new Set(
      [
        model.prototype.imageUrl,
        ...model.animations.flatMap((animation) =>
          animation.spriteSheet?.imageUrl
            ? [animation.spriteSheet.imageUrl]
            : [],
        ),
      ].filter(Boolean),
    );
    await Promise.all(
      [...urls].map(async (url) => {
        try {
          const texture = await Assets.load(url);
          texture.source.scaleMode = "nearest";
          this.unavailableTextureUrls.delete(url);
        } catch {
          this.unavailableTextureUrls.add(url);
        }
      }),
    );
    if (!this.destroyed) this.render();
  }

  private centerWorld() {
    this.viewport?.setZoom(INITIAL_SCALE);
    this.viewport?.moveCenter(650, 700);
  }

  private render() {
    if (!this.renderer || !this.viewport) return;
    this.renderer.render(this.scene.getSnapshot(), {
      ...this.props.model,
      unavailableTextureUrls: this.unavailableTextureUrls,
    });
    this.renderer.syncViewport(this.viewport, this.props.model.prototype);
  }

  private syncViewportGrid = () => {
    if (this.viewport && this.renderer)
      this.renderer.syncViewport(this.viewport, this.props.model.prototype);
  };

  private updateAnimation = () => {
    if (this.scene.getSnapshot().playing.size === 0) return;
    const now = performance.now();
    if (now - this.lastAnimationFrame < 160) return;
    this.lastAnimationFrame = now;
    this.scene.advanceAnimation(this.props.model);
    this.render();
  };
}
