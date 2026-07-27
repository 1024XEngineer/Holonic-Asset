import type { EditorCharacterAnimation } from "../../domain";
import type { CharacterCanvasNodeId } from "./character-node";

export type CanvasPosition = {
  x: number;
  y: number;
};

export const PROTOTYPE_NODE_ID = "prototype";

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
  return Object.fromEntries(
    getCanvasNodes(animations).map((node, index) => [
      node,
      {
        x: 80 + (index % 5) * 330,
        y: 220 + Math.floor(index / 5) * 370,
      },
    ]),
  );
}
