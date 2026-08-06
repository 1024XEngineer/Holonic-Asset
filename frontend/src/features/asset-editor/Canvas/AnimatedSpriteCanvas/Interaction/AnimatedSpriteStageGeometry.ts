import type { CharacterAnimation } from "@/model";
import { containsPoint } from "@/lib/rect";
import {
  getCanvasNodes,
  type CanvasPosition,
} from "../AnimatedSpriteCanvas.constants";
import {
  COLLAPSED_HEIGHT,
  EXPANDED_WIDTH,
  FRAME_GAP,
  FRAME_SIZE,
  getExpandedNodeHeight,
  NODE_WIDTH,
} from "../Runtime/AnimatedSpriteStage.constants";
import type {
  Bounds,
  AnimatedSpriteSceneSnapshot,
} from "../Runtime/AnimatedSpriteCanvas.types";
import {
  getAnimatedSpriteAnimation,
  type NodeId,
} from "../animated-sprite-node";

const FRAME_GRID_INSET = 8;
const FRAME_GRID_TOP = 48;
const CONTROL_HEIGHT = 32;
const CONTROL_BOTTOM = 8;
const PLAY_CONTROL = { x: 37, width: 68 } as const;
const EXPAND_CONTROL = { x: 113, width: 84 } as const;
type SpriteSheetShape = { columns: number; rows: number };

function isAnimationList(
  value: SpriteSheetShape | readonly CharacterAnimation[] | undefined,
): value is readonly CharacterAnimation[] {
  return Array.isArray(value);
}

export type AnimatedSpriteHitTarget =
  | { kind: "node"; node: NodeId }
  | { kind: "frame"; node: NodeId; index: number }
  | { kind: "frame-grid"; node: NodeId }
  | { kind: "play"; node: NodeId }
  | { kind: "expand"; node: NodeId };

export type AnimatedSpriteNodeLayout = {
  bounds: Bounds;
  frames: Bounds[];
  frameGrid?: Bounds;
  playControl?: Bounds;
  playEnabled: boolean;
  expandControl?: Bounds;
};

export function getFrameCount(
  node: NodeId,
  prototypeOrAnimations?:
    | { columns: number; rows: number }
    | readonly CharacterAnimation[],
  animations: readonly CharacterAnimation[] = [],
) {
  const prototype = isAnimationList(prototypeOrAnimations)
    ? undefined
    : prototypeOrAnimations;
  const resolvedAnimations = isAnimationList(prototypeOrAnimations)
    ? prototypeOrAnimations
    : animations;
  if (node === "prototype")
    return Math.max(1, (prototype?.columns ?? 1) * (prototype?.rows ?? 1));
  return Math.max(
    1,
    getAnimatedSpriteAnimation(node, resolvedAnimations)?.frameCount ?? 1,
  );
}

function getExpandedHeight(
  node: NodeId,
  prototype: SpriteSheetShape | undefined,
  animations: readonly CharacterAnimation[],
) {
  return getExpandedNodeHeight(getFrameCount(node, prototype, animations));
}

export function getNodeBounds(
  node: NodeId,
  position: CanvasPosition,
  expanded: boolean,
  prototypeOrAnimations?:
    | { columns: number; rows: number }
    | readonly CharacterAnimation[],
  animations: readonly CharacterAnimation[] = [],
): Bounds {
  const prototype = isAnimationList(prototypeOrAnimations)
    ? undefined
    : prototypeOrAnimations;
  const resolvedAnimations = isAnimationList(prototypeOrAnimations)
    ? prototypeOrAnimations
    : animations;
  return {
    ...position,
    width: expanded ? EXPANDED_WIDTH : NODE_WIDTH,
    height: expanded
      ? getExpandedHeight(node, prototype, resolvedAnimations)
      : COLLAPSED_HEIGHT,
  };
}

export function getAnimatedSpriteNodeLayout(
  node: NodeId,
  position: CanvasPosition,
  expanded: boolean,
  prototypeOrAnimations?:
    | { columns: number; rows: number }
    | readonly CharacterAnimation[],
  animations: readonly CharacterAnimation[] = [],
): AnimatedSpriteNodeLayout {
  const prototype = isAnimationList(prototypeOrAnimations)
    ? undefined
    : prototypeOrAnimations;
  const resolvedAnimations = isAnimationList(prototypeOrAnimations)
    ? prototypeOrAnimations
    : animations;
  const bounds = getNodeBounds(
    node,
    position,
    expanded,
    prototype,
    resolvedAnimations,
  );
  const frameCount = getFrameCount(node, prototype, resolvedAnimations);
  const frames = expanded
    ? Array.from({ length: frameCount }, (_, index) =>
        getFrameBounds(position, index),
      )
    : [];
  const controlsY = bounds.y + bounds.height - CONTROL_HEIGHT - CONTROL_BOTTOM;
  const animation = getAnimatedSpriteAnimation(node, resolvedAnimations);
  const hasControls = Boolean(animation);
  return {
    bounds,
    frames,
    frameGrid: expanded
      ? {
          x: bounds.x + FRAME_GRID_INSET,
          y: bounds.y + FRAME_GRID_TOP,
          width: EXPANDED_WIDTH - FRAME_GRID_INSET * 2,
          height:
            Math.ceil(frameCount / 4) * (FRAME_SIZE + FRAME_GAP) - FRAME_GAP,
        }
      : undefined,
    playControl: hasControls
      ? {
          x: bounds.x + PLAY_CONTROL.x,
          y: controlsY,
          width: PLAY_CONTROL.width,
          height: CONTROL_HEIGHT,
        }
      : undefined,
    playEnabled: hasControls && !expanded,
    expandControl: hasControls
      ? {
          x: bounds.x + EXPAND_CONTROL.x,
          y: controlsY,
          width: EXPAND_CONTROL.width,
          height: CONTROL_HEIGHT,
        }
      : undefined,
  };
}

export function getFrameBounds(
  position: CanvasPosition,
  index: number,
): Bounds {
  return {
    x: position.x + FRAME_GRID_INSET + (index % 4) * (FRAME_SIZE + FRAME_GAP),
    y:
      position.y +
      FRAME_GRID_TOP +
      Math.floor(index / 4) * (FRAME_SIZE + FRAME_GAP),
    width: FRAME_SIZE,
    height: FRAME_SIZE,
  };
}

export function hitTestAnimatedSpriteScene(
  scene: AnimatedSpriteSceneSnapshot,
  point: CanvasPosition,
  prototypeOrAnimations?:
    | { columns: number; rows: number }
    | readonly CharacterAnimation[],
  animations: readonly CharacterAnimation[] = [],
): AnimatedSpriteHitTarget | null {
  const prototype = Array.isArray(prototypeOrAnimations)
    ? undefined
    : prototypeOrAnimations;
  const resolvedAnimations = Array.isArray(prototypeOrAnimations)
    ? prototypeOrAnimations
    : animations;
  for (const node of getCanvasNodes(resolvedAnimations).reverse()) {
    const layout = getAnimatedSpriteNodeLayout(
      node,
      scene.positions[node],
      scene.expanded.has(node),
      prototype,
      resolvedAnimations,
    );
    for (let index = layout.frames.length - 1; index >= 0; index -= 1) {
      if (containsPoint(layout.frames[index], point))
        return { kind: "frame", node, index };
    }
    if (layout.frameGrid && containsPoint(layout.frameGrid, point))
      return { kind: "frame-grid", node };
    if (
      layout.playEnabled &&
      layout.playControl &&
      containsPoint(layout.playControl, point)
    )
      return { kind: "play", node };
    if (layout.expandControl && containsPoint(layout.expandControl, point))
      return { kind: "expand", node };
    if (containsPoint(layout.bounds, point)) return { kind: "node", node };
  }
  return null;
}
