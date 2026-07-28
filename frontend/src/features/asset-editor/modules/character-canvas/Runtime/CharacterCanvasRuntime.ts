import { Viewport } from "pixi-viewport";
import { Assets, Container } from "pixi.js";

import { getEditorCharacterAnimationClips } from "../../../domain";
import { CharacterStageInteraction } from "../Interaction/CharacterStageInteraction";
import { CharacterStageRenderer } from "../Renderer/CharacterStageRenderer";
import {
  getCharacterPixelScale,
  getCharacterMaxScale,
  INITIAL_SCALE,
  MIN_SCALE,
} from "./CharacterStage.constants";
import type {
  CharacterCanvasRuntimeProps,
  CharacterStageContext,
} from "./CharacterCanvas.types";
import { CharacterScene } from "./CharacterScene";
import { StageRuntime } from "./StageRuntime";

export class CharacterCanvasRuntime {
  private readonly runtime = new StageRuntime();
  private interaction?: CharacterStageInteraction;
  private resizeObserver?: ResizeObserver;
  private renderer?: CharacterStageRenderer;
  private viewport?: Viewport;
  private props: CharacterCanvasRuntimeProps;
  private lastAnimationFrame = performance.now();
  private readonly unavailableTextureUrls = new Set<string>();
  private readonly scene: CharacterScene;

  constructor(props: CharacterCanvasRuntimeProps) {
    this.props = props;
    this.scene = new CharacterScene(props.model);
  }

  async initialize(host: HTMLElement) {
    await this.preloadCharacterTextures();
    await this.runtime.initialize(host);
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
        maxScale: getCharacterMaxScale(this.props.model.prototype),
      });
    app.stage.addChild(viewport);
    this.viewport = viewport;
    const world = new Container();
    viewport.addChild(world);
    this.renderer = new CharacterStageRenderer(app.stage, world);
    viewport.on("moved", this.syncViewportGrid);
    viewport.on("zoomed", this.syncViewportGrid);

    const context: CharacterStageContext = {
      viewport,
      actions: {
        onSelect: (node) => this.props.actions.onSelect(node),
        onSelectFrame: (node, index) =>
          this.props.actions.onSelectFrame(node, index),
        onSelectFrames: (node, indexes) =>
          this.props.actions.onSelectFrames(node, indexes),
        onSelectNodes: (nodes) => this.props.actions.onSelectNodes(nodes),
        onClearSelection: () => this.props.actions.onClearSelection(),
        onNodePositionChange: (node, position) =>
          this.props.actions.onNodePositionChange(node, position),
        onSwitchDirection: (node, direction) =>
          this.props.actions.onSwitchDirection(node, direction),
      },
      getScene: () => this.scene.getSnapshot(),
      getAnimations: () => this.props.model.animations,
      moveNode: (node, position) => this.scene.moveNode(node, position),
      setMarquee: (marquee) => this.scene.setMarquee(marquee),
      getDragStep: () => getCharacterPixelScale(this.props.model.prototype),
      toggleExpanded: (node) => this.scene.toggleExpanded(node),
      togglePlaying: (node) => this.scene.togglePlaying(node),
      switchDirection: (node) => {
        const change = this.scene.switchDirection(node, this.props.model);
        if (change)
          this.props.actions.onSwitchDirection(
            change.nodeId,
            change.directionId,
          );
      },
      getActiveDirections: () => this.scene.getSnapshot().activeDirections,
      render: () => this.render(),
    };
    this.interaction = new CharacterStageInteraction(app.canvas, context);
    this.resizeObserver = new ResizeObserver(() => {
      viewport.resize(app.screen.width, app.screen.height);
      this.render();
    });
    this.resizeObserver.observe(host);
    app.ticker.add(this.updateAnimation);
    this.centerWorld();
    this.syncProps(this.props);
  }

  syncProps(props: CharacterCanvasRuntimeProps) {
    this.props = props;
    this.scene.synchronize(props.model);
    this.render();
  }

  private async preloadCharacterTextures() {
    const urls = new Set(
      [
        this.props.model.prototype.imageUrl,
        ...getEditorCharacterAnimationClips(
          this.props.model.animations,
        ).flatMap((animation) =>
          animation.spriteSheet ? [animation.spriteSheet.imageUrl] : [],
        ),
      ].filter(Boolean),
    );
    await Promise.all(
      [...urls].map(async (url) => {
        try {
          const texture = await Assets.load(url);
          texture.source.scaleMode = "nearest";
        } catch {
          this.unavailableTextureUrls.add(url);
        }
      }),
    );
  }

  destroy() {
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

  private centerWorld() {
    this.viewport?.setZoom(INITIAL_SCALE);
    this.viewport?.moveCenter(650, 700);
  }

  private render() {
    this.renderer?.render(this.scene.getSnapshot(), {
      ...this.props.model,
      unavailableTextureUrls: this.unavailableTextureUrls,
    });
    this.syncViewportGrid();
  }

  private syncViewportGrid = () => {
    if (!this.viewport || !this.renderer) return;
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
