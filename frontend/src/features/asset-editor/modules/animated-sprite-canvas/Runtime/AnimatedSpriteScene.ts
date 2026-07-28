import { isEditorCharacterAnimationGroup } from "../../../domain";
import type { AnimatedSpriteCanvasModel } from "../AnimatedSpriteCanvas.interface";
import {
  createDefaultAnimatedSpriteDirections,
  findAnimatedSpriteAnimationGroup,
  getAnimatedSpriteNodeId,
  getPreferredAnimatedSpriteDirection,
  type NodeId,
} from "../animated-sprite-node";
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
      activeDirections: new Map(
        Object.entries(createDefaultAnimatedSpriteDirections(model.animations)),
      ),
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
      [...this.state.expanded]
        .map((node) => getAnimatedSpriteNodeId(node, model.animations))
        .filter((node) => canvasNodes.has(node)),
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
      const node = getAnimatedSpriteNodeId(frame.nodeId, model.animations);
      if (canvasNodes.has(node)) this.state.expanded.add(node);
    }

    const activeDirections = new Map<NodeId, NodeId>();
    for (const animation of model.animations) {
      if (!isEditorCharacterAnimationGroup(animation)) continue;
      const requested = model.activeDirections?.[animation.id];
      const current = this.state.activeDirections.get(animation.id);
      const direction = getPreferredAnimatedSpriteDirection(
        animation,
        requested,
        current,
      );
      const directionId = direction?.id ?? animation.directions[0].id;
      activeDirections.set(animation.id, directionId);
      if (current !== directionId)
        this.state.previewFrames.set(animation.id, 0);
    }
    this.state.activeDirections = activeDirections;

    for (const node of this.state.playing) {
      const frameCount = getFrameCount(
        node,
        model.animations,
        this.state.activeDirections,
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

  switchDirection(node: NodeId, model: AnimatedSpriteCanvasModel) {
    const group = findAnimatedSpriteAnimationGroup(node, model.animations);
    if (!group || group.directions.length < 2) return;
    const current = this.state.activeDirections.get(group.id);
    const index = Math.max(
      0,
      group.directions.findIndex((direction) => direction.id === current),
    );
    const direction = group.directions[(index + 1) % group.directions.length];
    this.state.activeDirections.set(group.id, direction.id);
    this.state.previewFrames.set(group.id, 0);

    return { nodeId: group.id, directionId: direction.id };
  }

  advanceAnimation(model: AnimatedSpriteCanvasModel) {
    for (const node of this.state.playing) {
      this.state.previewFrames.set(
        node,
        ((this.state.previewFrames.get(node) ?? 0) + 1) %
          getFrameCount(node, model.animations, this.state.activeDirections),
      );
    }
  }
}
