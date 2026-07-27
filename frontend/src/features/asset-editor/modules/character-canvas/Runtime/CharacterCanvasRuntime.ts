import { Viewport } from "pixi-viewport";
import { Assets, Container } from "pixi.js";

import {
  getEditorCharacterAnimationClips,
  isEditorCharacterAnimationGroup,
} from "../../../domain";
import {
  createDefaultCharacterDirections,
  findCharacterAnimationGroup,
  getPreferredCharacterDirection,
  type NodeId,
} from "../character-node";
import { createDefaultCanvasPositions } from "../CharacterCanvas.constants";
import { CharacterStageInteraction } from "../Interaction/CharacterStageInteraction";
import { getFrameCount } from "../Interaction/CharacterStageGeometry";
import { CharacterStageRenderer } from "../Renderer/CharacterStageRenderer";
import {
  getCharacterPixelScale,
  getCharacterMaxScale,
  INITIAL_SCALE,
  MIN_SCALE,
  WORLD_HEIGHT,
  WORLD_WIDTH,
} from "./CharacterStage.constants";
import type {
  CharacterCanvasRuntimeProps,
  CharacterSceneState,
  CharacterStageContext,
} from "./CharacterCanvas.types";
import { StageRuntime } from "./StageRuntime";

export class CharacterCanvasRuntime {
  private readonly runtime = new StageRuntime();
  private interaction?: CharacterStageInteraction;
  private resizeObserver?: ResizeObserver;
  private renderer?: CharacterStageRenderer;
  private viewport?: Viewport;
  private props: CharacterCanvasRuntimeProps;
  private lastAnimationFrame = performance.now();

  private readonly state: CharacterSceneState;

  constructor(props: CharacterCanvasRuntimeProps) {
    this.props = props;
    this.state = {
      positions: createDefaultCanvasPositions(props.model.animations),
      expanded: new Set(),
      playing: new Set(),
      previewFrames: new Map(),
      activeDirections: new Map(
        Object.entries(
          createDefaultCharacterDirections(props.model.animations),
        ),
      ),
      marquee: null,
    };
  }

  async initialize(host: HTMLElement) {
    await this.preloadCharacterTextures();
    await this.runtime.initialize(host);
    const { app } = this.runtime;
    const viewport = new Viewport({
      screenWidth: app.screen.width,
      screenHeight: app.screen.height,
      worldWidth: WORLD_WIDTH,
      worldHeight: WORLD_HEIGHT,
      events: app.renderer.events,
      ticker: app.ticker,
    });
    viewport.eventMode = "static";
    viewport
      .drag({ mouseButtons: "middle" })
      .wheel()
      .clamp({ direction: "all", underflow: "center" })
      .clampZoom({
        minScale: MIN_SCALE,
        maxScale: getCharacterMaxScale(this.props.model.prototype),
      });
    app.stage.addChild(viewport);
    this.viewport = viewport;
    const world = new Container();
    viewport.addChild(world);
    this.renderer = new CharacterStageRenderer(world);

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
      getScene: () => this.state,
      getAnimations: () => this.props.model.animations,
      moveNode: (node, position) => {
        this.state.positions[node] = position;
      },
      setMarquee: (marquee) => {
        this.state.marquee = marquee;
      },
      getDragStep: () => getCharacterPixelScale(this.props.model.prototype),
      toggleExpanded: (node) => {
        this.state.playing.delete(node);
        if (this.state.expanded.has(node)) this.state.expanded.delete(node);
        else this.state.expanded.add(node);
      },
      togglePlaying: (node) => {
        if (this.state.expanded.has(node)) return;
        if (this.state.playing.has(node)) this.state.playing.delete(node);
        else this.state.playing.add(node);
      },
      switchDirection: (node) => {
        const group = findCharacterAnimationGroup(
          node,
          this.props.model.animations,
        );
        if (!group || group.directions.length < 2) return;
        const current = this.state.activeDirections.get(group.id);
        const index = Math.max(
          0,
          group.directions.findIndex((direction) => direction.id === current),
        );
        const direction =
          group.directions[(index + 1) % group.directions.length];
        this.state.activeDirections.set(group.id, direction.id);
        this.state.previewFrames.set(group.id, 0);
        this.props.actions.onSwitchDirection(group.id, direction.id);
      },
      getActiveDirections: () => this.state.activeDirections,
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
    this.state.positions = createDefaultCanvasPositions(props.model.animations);
    for (const [node, position] of Object.entries(
      props.model.nodePositions ?? {},
    )) {
      this.state.positions[node as NodeId] = { ...position };
    }
    for (const frame of props.model.selection.frames)
      this.state.expanded.add(frame.nodeId);
    for (const animation of props.model.animations) {
      if (!isEditorCharacterAnimationGroup(animation)) continue;
      const requested = props.model.activeDirections?.[animation.id];
      const current = this.state.activeDirections.get(animation.id);
      const direction = getPreferredCharacterDirection(
        animation,
        requested,
        current,
      );
      this.state.activeDirections.set(
        animation.id,
        direction?.id ?? animation.directions[0].id,
      );
    }
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
    const textures = await Promise.all(
      [...urls].map((url) => Assets.load(url)),
    );
    textures.forEach((texture) => {
      texture.source.scaleMode = "nearest";
    });
  }

  destroy() {
    this.runtime.app.ticker.remove(this.updateAnimation);
    this.resizeObserver?.disconnect();
    this.interaction?.destroy();
    this.viewport?.removeFromParent();
    this.viewport?.destroy({ children: true });
    this.viewport = undefined;
    this.runtime.destroy();
  }

  private centerWorld() {
    this.viewport?.setZoom(INITIAL_SCALE);
    this.viewport?.moveCenter(650, WORLD_HEIGHT / 2);
  }

  private render() {
    this.renderer?.render(this.state, this.props.model);
  }

  private updateAnimation = () => {
    if (this.state.playing.size === 0) return;
    const now = performance.now();
    if (now - this.lastAnimationFrame < 160) return;
    this.lastAnimationFrame = now;
    for (const node of this.state.playing) {
      this.state.previewFrames.set(
        node,
        ((this.state.previewFrames.get(node) ?? 0) + 1) %
          getFrameCount(
            node,
            this.props.model.animations,
            this.state.activeDirections,
          ),
      );
    }
    this.render();
  };
}
