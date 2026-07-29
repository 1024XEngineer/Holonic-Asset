import type { AnimatedSpriteCanvasModel } from "../AnimatedSpriteCanvas.interface";
import type { NodeId } from "../animated-sprite-node";
import {
  createDefaultCanvasPositions,
  getCanvasNodes,
  type CanvasPosition,
} from "../AnimatedSpriteCanvas.constants";
import { getFrameCount } from "../Interaction/AnimatedSpriteStageGeometry";
import type {
  AnimatedSpriteSceneSnapshot,
  AnimatedSpriteSceneState,
} from "./AnimatedSpriteCanvas.types";

export class AnimatedSpriteScene {
  private readonly state: AnimatedSpriteSceneState;

  constructor(model: AnimatedSpriteCanvasModel) {
    this.state = {
      positions: createDefaultCanvasPositions(model.animations),
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
    this.state.positions = createDefaultCanvasPositions(model.animations);
    const canvasNodes = new Set(getCanvasNodes(model.animations));
    this.state.expanded = new Set(
      [...this.state.expanded].filter((node) => canvasNodes.has(node)),
    );
    this.state.playing = new Set(
      [...this.state.playing].filter((node) => canvasNodes.has(node)),
    );
    this.state.previewFrames = new Map(
      [...this.state.previewFrames].filter(([node]) => canvasNodes.has(node)),
    );
    for (const [node, position] of Object.entries(model.nodePositions ?? {})) {
      this.state.positions[node as NodeId] = { ...position };
    }
    for (const frame of model.selection.frames) {
      const node = frame.nodeId;
      if (canvasNodes.has(node)) this.state.expanded.add(node);
    }

    for (const node of this.state.playing) {
      const frameCount = getFrameCount(node, model.animations);
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
      this.state.previewFrames.set(
        node,
        ((this.state.previewFrames.get(node) ?? 0) + 1) %
          getFrameCount(node, model.animations),
      );
    }
  }
}
