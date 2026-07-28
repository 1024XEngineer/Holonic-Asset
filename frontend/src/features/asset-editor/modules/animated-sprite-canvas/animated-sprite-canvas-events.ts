import type { AnimatedSpriteCanvasActions } from "./Runtime/AnimatedSpriteCanvas.types";
import type {
  AnimatedSpriteCanvasEvent,
  AnimatedSpriteCanvasSelection,
} from "./AnimatedSpriteCanvas.interface";

export function createAnimatedSpriteCanvasActions(
  onEvent: (event: AnimatedSpriteCanvasEvent) => void,
): AnimatedSpriteCanvasActions {
  const changeSelection = (selection: AnimatedSpriteCanvasSelection) => {
    onEvent({ type: "selection.changed", selection });
  };

  return {
    onSelect: (nodeId) => changeSelection({ nodeIds: [nodeId], frames: [] }),
    onSelectFrame: (nodeId, index) =>
      changeSelection({
        nodeIds: [nodeId],
        frames: [{ nodeId, index }],
      }),
    onSelectFrames: (nodeId, indexes) =>
      changeSelection({
        nodeIds: [nodeId],
        frames: indexes.map((index) => ({ nodeId, index })),
      }),
    onSelectNodes: (nodeIds) => changeSelection({ nodeIds, frames: [] }),
    onClearSelection: () => changeSelection({ nodeIds: [], frames: [] }),
    onNodePositionChange: (nodeId, position) => {
      onEvent({ type: "node-position.committed", nodeId, position });
    },
    onSwitchDirection: (nodeId, directionId) => {
      onEvent({ type: "direction.changed", nodeId, directionId });
    },
  };
}
