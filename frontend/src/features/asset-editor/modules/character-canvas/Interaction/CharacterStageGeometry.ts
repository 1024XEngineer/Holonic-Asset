import type { EditorCharacterAnimation } from "../../../domain";
import {
  getCharacterCanvasAnimation,
  type CharacterDirectionMap,
  type NodeId,
} from "../character-node";
import {
  getCanvasNodes,
  type CanvasPosition,
} from "../CharacterCanvas.constants";
import {
  COLLAPSED_HEIGHT,
  EXPANDED_WIDTH,
  FRAME_GAP,
  FRAME_SIZE,
  NODE_WIDTH,
} from "../Runtime/CharacterStage.constants";
import type {
  Bounds,
  CharacterSceneSnapshot,
} from "../Runtime/CharacterCanvas.types";

const FRAME_GRID_INSET = 8;
const FRAME_GRID_TOP = 48;
const CONTROL_HEIGHT = 32;
const CONTROL_BOTTOM = 8;
const PLAY_CONTROL = { x: 37, width: 68 } as const;
const EXPAND_CONTROL = { x: 113, width: 84 } as const;
const MULTI_PLAY_CONTROL = { x: 8, width: 64 } as const;
const SWITCH_CONTROL = { x: 76, width: 72 } as const;
const MULTI_EXPAND_CONTROL = { x: 156, width: 60 } as const;

type CharacterHitTarget =
  | { kind: "node"; node: NodeId }
  | { kind: "frame"; node: NodeId; index: number }
  | { kind: "frame-grid"; node: NodeId }
  | { kind: "play"; node: NodeId }
  | { kind: "expand"; node: NodeId }
  | { kind: "switch"; node: NodeId };

type CharacterNodeLayout = {
  bounds: Bounds;
  frames: Bounds[];
  frameGrid?: Bounds;
  playControl?: Bounds;
  playEnabled: boolean;
  expandControl?: Bounds;
  switchControl?: Bounds;
};

export function getFrameCount(
  node: NodeId,
  animations: EditorCharacterAnimation[] = [],
  directions?: CharacterDirectionMap,
) {
  return (
    getCharacterCanvasAnimation(node, animations, directions)?.frameCount ?? 1
  );
}

function getExpandedHeight(
  node: NodeId,
  animations: EditorCharacterAnimation[],
  directions?: CharacterDirectionMap,
) {
  return (
    48 +
    Math.ceil(getFrameCount(node, animations, directions) / 4) *
      (FRAME_SIZE + FRAME_GAP) +
    48
  );
}

export function getNodeBounds(
  node: NodeId,
  position: CanvasPosition,
  expanded: boolean,
  animations: EditorCharacterAnimation[] = [],
  directions?: CharacterDirectionMap,
): Bounds {
  return {
    ...position,
    width: expanded ? EXPANDED_WIDTH : NODE_WIDTH,
    height: expanded
      ? getExpandedHeight(node, animations, directions)
      : COLLAPSED_HEIGHT,
  };
}

export function getCharacterNodeLayout(
  node: NodeId,
  position: CanvasPosition,
  expanded: boolean,
  animations: EditorCharacterAnimation[] = [],
  directions?: CharacterDirectionMap,
): CharacterNodeLayout {
  const bounds = getNodeBounds(
    node,
    position,
    expanded,
    animations,
    directions,
  );
  const frames = expanded
    ? Array.from(
        { length: getFrameCount(node, animations, directions) },
        (_, index) => getFrameBounds(position, index),
      )
    : [];
  const controlsY = bounds.y + bounds.height - CONTROL_HEIGHT - CONTROL_BOTTOM;
  const animation = getCharacterCanvasAnimation(node, animations, directions);
  const hasControls = Boolean(animation);
  const group = animations.find((candidate) => candidate.id === node);
  const hasSwitch = Boolean(
    group && "directions" in group && group.directions.length > 1,
  );

  return {
    bounds,
    frames,
    frameGrid: expanded
      ? {
          x: bounds.x + FRAME_GRID_INSET,
          y: bounds.y + FRAME_GRID_TOP,
          width: EXPANDED_WIDTH - FRAME_GRID_INSET * 2,
          height:
            Math.ceil(getFrameCount(node, animations, directions) / 4) *
              (FRAME_SIZE + FRAME_GAP) -
            FRAME_GAP,
        }
      : undefined,
    playControl: hasControls
      ? {
          x: bounds.x + (hasSwitch ? MULTI_PLAY_CONTROL.x : PLAY_CONTROL.x),
          y: controlsY,
          width: hasSwitch ? MULTI_PLAY_CONTROL.width : PLAY_CONTROL.width,
          height: CONTROL_HEIGHT,
        }
      : undefined,
    playEnabled: hasControls && !expanded,
    expandControl: hasControls
      ? {
          x: bounds.x + (hasSwitch ? MULTI_EXPAND_CONTROL.x : EXPAND_CONTROL.x),
          y: controlsY,
          width: hasSwitch ? MULTI_EXPAND_CONTROL.width : EXPAND_CONTROL.width,
          height: CONTROL_HEIGHT,
        }
      : undefined,
    switchControl: hasSwitch
      ? {
          x: bounds.x + SWITCH_CONTROL.x,
          y: controlsY,
          width: SWITCH_CONTROL.width,
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

export function hitTestCharacterScene(
  scene: CharacterSceneSnapshot,
  point: CanvasPosition,
  animations: EditorCharacterAnimation[] = [],
  directions?: CharacterDirectionMap,
): CharacterHitTarget | null {
  for (const node of getCanvasNodes(animations).reverse()) {
    const layout = getCharacterNodeLayout(
      node,
      scene.positions[node],
      scene.expanded.has(node),
      animations,
      directions ?? scene.activeDirections,
    );

    for (let index = layout.frames.length - 1; index >= 0; index -= 1) {
      if (contains(layout.frames[index], point))
        return { kind: "frame", node, index };
    }
    if (layout.frameGrid && contains(layout.frameGrid, point))
      return { kind: "frame-grid", node };
    if (
      layout.playEnabled &&
      layout.playControl &&
      contains(layout.playControl, point)
    )
      return { kind: "play", node };
    if (layout.switchControl && contains(layout.switchControl, point))
      return { kind: "switch", node };
    if (layout.expandControl && contains(layout.expandControl, point))
      return { kind: "expand", node };
    if (contains(layout.bounds, point)) return { kind: "node", node };
  }

  return null;
}

function contains(bounds: Bounds, point: CanvasPosition) {
  return (
    point.x >= bounds.x &&
    point.x <= bounds.x + bounds.width &&
    point.y >= bounds.y &&
    point.y <= bounds.y + bounds.height
  );
}

export function intersects(left: Bounds, right: Bounds) {
  return (
    left.x < right.x + right.width &&
    left.x + left.width > right.x &&
    left.y < right.y + right.height &&
    left.y + left.height > right.y
  );
}

export function normalizeBounds(
  start: CanvasPosition,
  end: CanvasPosition,
): Bounds {
  return {
    x: Math.min(start.x, end.x),
    y: Math.min(start.y, end.y),
    width: Math.abs(end.x - start.x),
    height: Math.abs(end.y - start.y),
  };
}
