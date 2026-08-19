import type { AnimatedSpriteCanvasModel } from "../AnimatedSpriteCanvas.interface";
import type { NodeId } from "../animated-sprite-node";
import {
  createDefaultCanvasPositions,
  getCanvasNodes,
  type CanvasPosition,
} from "../AnimatedSpriteCanvas.constants";
import { getAnimatedSpriteFrameCount } from "../animated-sprite-frame-count";
import type {
  AnimatedSpriteSceneSnapshot,
  AnimatedSpriteSceneState,
} from "./AnimatedSpriteCanvas.types";

export class AnimatedSpriteScene {
  private readonly state: AnimatedSpriteSceneState;

  constructor(model: AnimatedSpriteCanvasModel) {
    this.state = {
      positions: {},
      expanded: new Set(),
      playing: new Set(),
      previewFrames: new Map(),
      marquee: null,
    };
    this.synchronize(model);
  }

  getSnapshot(): AnimatedSpriteSceneSnapshot {
    return this.state;
  }

  synchronize(model: AnimatedSpriteCanvasModel) {
    const nodes = getCanvasNodes(model.animations);
    const defaults = createDefaultCanvasPositions(model.animations);
    const previous = this.state.positions;
    this.state.positions = Object.fromEntries(
      nodes.map((node) => [
        node,
        {
          ...(model.nodePositions?.[node] ?? previous[node] ?? defaults[node]),
        },
      ]),
    ) as Record<NodeId, CanvasPosition>;
    const canvasNodes = new Set(nodes);
    this.state.expanded = new Set(
      [...this.state.expanded].filter((node) => canvasNodes.has(node)),
    );
    this.state.playing = new Set(
      [...this.state.playing].filter((node) => canvasNodes.has(node)),
    );
    this.state.previewFrames = new Map(
      [...this.state.previewFrames].filter(([node]) => canvasNodes.has(node)),
    );
    for (const frame of model.selection.frames) {
      if (canvasNodes.has(frame.nodeId)) this.state.expanded.add(frame.nodeId);
    }
    for (const node of this.state.playing) {
      const frameCount = getAnimatedSpriteFrameCount(
        node,
        model.prototype,
        model.animations,
      );
      this.state.previewFrames.set(
        node,
        (this.state.previewFrames.get(node) ?? 0) % frameCount,
      );
    }
  }

  moveNode(node: NodeId, position: CanvasPosition) {
    this.state.positions[node] = position;
  }
  setMarquee(marquee: AnimatedSpriteSceneState["marquee"]) {
    this.state.marquee = marquee;
  }
  toggleExpanded(node: NodeId) {
    this.state.playing.delete(node);
    if (this.state.expanded.has(node)) this.state.expanded.delete(node);
    else this.state.expanded.add(node);
  }
  togglePlaying(node: NodeId) {
    if (this.state.expanded.has(node)) return;
    if (this.state.playing.has(node)) this.state.playing.delete(node);
    else this.state.playing.add(node);
  }
  advanceAnimation(model: AnimatedSpriteCanvasModel) {
    for (const node of this.state.playing) {
      const frameCount = getAnimatedSpriteFrameCount(
        node,
        model.prototype,
        model.animations,
      );
      this.state.previewFrames.set(
        node,
        ((this.state.previewFrames.get(node) ?? 0) + 1) % frameCount,
      );
    }
  }
}

export function hasAnimatedSpriteCanvasModelChanged(
  previous: AnimatedSpriteCanvasModel,
  next: AnimatedSpriteCanvasModel,
) {
  return (
    previous.prototype !== next.prototype ||
    previous.animations !== next.animations ||
    previous.nodePositions !== next.nodePositions ||
    previous.selection !== next.selection ||
    previous.review !== next.review
  );
}
