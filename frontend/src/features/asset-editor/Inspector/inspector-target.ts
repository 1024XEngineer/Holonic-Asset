import type { CharacterAnimation, CharacterSpriteSheet } from "@/model";

import {
  getAnimatedSpriteNodeLabel,
  type AnimatedSpriteNodeId,
} from "../Canvas/AnimatedSpriteCanvas";
import { resolveSpriteSheetFrame } from "../Canvas/AnimatedSpriteCanvas/sprite-sheet-grid";
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
    const resolvedFrame = resolveSpriteSheetFrame(
      spriteSheet,
      selectedFrame?.index ?? 0,
    );
    return {
      label: nodeId
        ? `${getAnimatedSpriteNodeLabel(nodeId, animations)} - ${frameLabel} ${frames}`
        : `${frameLabel} ${frames}`,
      detail: "Selected on canvas",
      thumbnail: {
        imageUrl: resolvedFrame.imageUrl,
        column: resolvedFrame.column,
        row: resolvedFrame.row,
        columns: resolvedFrame.columns,
        rows: resolvedFrame.rows,
      },
    };
  }

  if (selectedNodes.length > 0) {
    const nodeId = selectedNodes[0] ?? "prototype";
    const spriteSheet = getSpriteSheet(nodeId, animations, prototype);
    const resolvedFrame = resolveSpriteSheetFrame(spriteSheet, 0);
    return {
      label: selectedNodes
        .map((nodeId) => getAnimatedSpriteNodeLabel(nodeId, animations))
        .join(", "),
      detail: "Selected item",
      thumbnail: {
        imageUrl: resolvedFrame.imageUrl,
        column: resolvedFrame.column,
        row: resolvedFrame.row,
        columns: resolvedFrame.columns,
        rows: resolvedFrame.rows,
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
