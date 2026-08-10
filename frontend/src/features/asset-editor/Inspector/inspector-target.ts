import type { CharacterAnimation } from "@/model";

import {
  getAnimatedSpriteNodeLabel,
  type AnimatedSpriteNodeId,
} from "../Canvas/AnimatedSpriteCanvas";
import type {
  InspectorFrameSelection,
  InspectorTargetSummary,
} from "./inspector.types";

export function getInspectorTargetSummary(
  selectedNodes: AnimatedSpriteNodeId[],
  selectedFrames: InspectorFrameSelection[],
  animations: CharacterAnimation[],
): InspectorTargetSummary {
  if (selectedFrames.length > 0) {
    const nodeId = selectedFrames[0]?.nodeId;
    const frames = selectedFrames.map((frame) => frame.index + 1).join(", ");
    return {
      label: nodeId
        ? getAnimatedSpriteNodeLabel(nodeId, animations)
        : "Selected frames",
      detail: `Frames ${frames}`,
    };
  }

  if (selectedNodes.length > 0) {
    return {
      label: selectedNodes
        .map((nodeId) => getAnimatedSpriteNodeLabel(nodeId, animations))
        .join(", "),
      detail: "Selected item",
    };
  }

  return {
    label: "Entire asset",
    detail: "Prompt applies to the complete asset",
  };
}
