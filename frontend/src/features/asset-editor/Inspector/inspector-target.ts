import type { CharacterAnimation, CharacterSpriteSheet } from "@/model";

import {
  getAnimatedSpriteNodeLabel,
  type AnimatedSpriteNodeId,
} from "../Canvas/AnimatedSpriteCanvas";
import { getSpriteSheetFramePosition } from "../Canvas/AnimatedSpriteCanvas/sprite-sheet-grid";
import type {
  InspectorFrameSelection,
  InspectorTargetSummary,
} from "./inspector.types";

export function getInspectorTargetSummary(
  selectedNodes: AnimatedSpriteNodeId[],
  selectedFrames: InspectorFrameSelection[],
  animations: CharacterAnimation[],
  prototype: CharacterSpriteSheet,
): InspectorTargetSummary | null {
  if (selectedFrames.length > 0) {
    const nodeId = selectedFrames[0]?.nodeId;
    const selectedFrame = selectedFrames[0];
    const frames = selectedFrames.map((frame) => frame.index + 1).join(", ");
    const frameLabel = selectedFrames.length === 1 ? "Frame" : "Frames";
    const spriteSheet = getSpriteSheet(nodeId, animations, prototype);
    const position = getSpriteSheetFramePosition(
      selectedFrame?.index ?? 0,
      spriteSheet,
    );
    return {
      label: nodeId
        ? `${getAnimatedSpriteNodeLabel(nodeId, animations)} - ${frameLabel} ${frames}`
        : `${frameLabel} ${frames}`,
      detail: "Selected on canvas",
      thumbnail: {
        imageUrl: spriteSheet.imageUrl,
        ...position,
        columns: spriteSheet.columns,
        rows: spriteSheet.rows,
      },
    };
  }

  if (selectedNodes.length > 0) {
    const nodeId = selectedNodes[0] ?? "prototype";
    const spriteSheet = getSpriteSheet(nodeId, animations, prototype);
    return {
      label: selectedNodes
        .map((nodeId) => getAnimatedSpriteNodeLabel(nodeId, animations))
        .join(", "),
      detail: "Selected item",
      thumbnail: {
        imageUrl: spriteSheet.imageUrl,
        column: 0,
        row: spriteSheet.row ?? 0,
        columns: spriteSheet.columns,
        rows: spriteSheet.rows,
      },
    };
  }

  return null;
}

function getSpriteSheet(
  nodeId: AnimatedSpriteNodeId | undefined,
  animations: CharacterAnimation[],
  prototype: CharacterSpriteSheet,
) {
  return nodeId === "prototype"
    ? prototype
    : (animations.find((animation) => animation.id === nodeId)?.spriteSheet ??
        prototype);
}
