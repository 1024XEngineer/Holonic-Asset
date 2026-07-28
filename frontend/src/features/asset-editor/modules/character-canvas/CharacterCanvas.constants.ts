import {
  isEditorCharacterAnimationGroup,
  type EditorCharacterAnimation,
} from "../../domain";
import type { CharacterCanvasNodeId } from "./character-node";
import {
  COLLAPSED_HEIGHT,
  EXPANDED_WIDTH,
  getExpandedNodeHeight,
} from "./Runtime/CharacterStage.constants";

export type CanvasPosition = {
  x: number;
  y: number;
};

export const PROTOTYPE_NODE_ID = "prototype";

const DEFAULT_LAYOUT_COLUMNS = 4;
const DEFAULT_LAYOUT_GAP = 64;
const DEFAULT_LAYOUT_START = { x: 80, y: 220 };

export function getCanvasNodes(
  animations: EditorCharacterAnimation[],
): CharacterCanvasNodeId[] {
  return [
    PROTOTYPE_NODE_ID,
    ...animations
      .map((animation) => animation.id)
      .filter((id) => id !== PROTOTYPE_NODE_ID),
  ];
}

export function createDefaultCanvasPositions(
  animations: EditorCharacterAnimation[],
): Record<CharacterCanvasNodeId, CanvasPosition> {
  const nodes = getCanvasNodes(animations);
  const positions: Record<CharacterCanvasNodeId, CanvasPosition> = {};
  let y = DEFAULT_LAYOUT_START.y;

  for (
    let rowStart = 0;
    rowStart < nodes.length;
    rowStart += DEFAULT_LAYOUT_COLUMNS
  ) {
    const row = nodes.slice(rowStart, rowStart + DEFAULT_LAYOUT_COLUMNS);
    const rowHeight = Math.max(
      ...row.map((node) => getDefaultNodeHeight(node, animations)),
    );

    row.forEach((node, column) => {
      positions[node] = {
        x:
          DEFAULT_LAYOUT_START.x +
          column * (EXPANDED_WIDTH + DEFAULT_LAYOUT_GAP),
        y,
      };
    });
    y += rowHeight + DEFAULT_LAYOUT_GAP;
  }

  return positions;
}

function getDefaultNodeHeight(
  node: CharacterCanvasNodeId,
  animations: EditorCharacterAnimation[],
) {
  if (node === PROTOTYPE_NODE_ID) return COLLAPSED_HEIGHT;
  const animation = animations.find((candidate) => candidate.id === node);
  if (!animation) return COLLAPSED_HEIGHT;
  const frameCount = isEditorCharacterAnimationGroup(animation)
    ? Math.max(...animation.directions.map((direction) => direction.frameCount))
    : animation.frameCount;
  return Math.max(COLLAPSED_HEIGHT, getExpandedNodeHeight(frameCount));
}
