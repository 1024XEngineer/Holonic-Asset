import type { AnimatedSpriteCanvasActions } from "./Runtime/AnimatedSpriteCanvas.types";
import type {
  AnimatedSpriteCanvasEvent,
  AnimatedSpriteCanvasFrameSelection,
  AnimatedSpriteCanvasSelection,
} from "./AnimatedSpriteCanvas.interface";

export function createAnimatedSpriteCanvasActions(
  onEvent: (event: AnimatedSpriteCanvasEvent) => void,
  currentSelection: AnimatedSpriteCanvasSelection = {
    nodeIds: [],
    frames: [],
  },
): AnimatedSpriteCanvasActions {
  const changeSelection = (selection: AnimatedSpriteCanvasSelection) =>
    onEvent({ type: "selection.changed", selection });
  const toggleNodes = (nodeIds: string[]): AnimatedSpriteCanvasSelection => {
    const targets = new Set(nodeIds);
    const removed = new Set(
      currentSelection.nodeIds.filter((nodeId) => targets.has(nodeId)),
    );
    return {
      nodeIds: [
        ...currentSelection.nodeIds.filter((nodeId) => !removed.has(nodeId)),
        ...nodeIds.filter(
          (nodeId) => !currentSelection.nodeIds.includes(nodeId),
        ),
      ],
      frames: currentSelection.frames.filter(
        ({ nodeId }) => !removed.has(nodeId),
      ),
    };
  };
  const toggleFrames = (
    frames: AnimatedSpriteCanvasFrameSelection[],
  ): AnimatedSpriteCanvasSelection => {
    const nextFrames = [...currentSelection.frames];
    const touchedNodes = new Set<string>();
    for (const frame of frames) {
      touchedNodes.add(frame.nodeId);
      const selectedIndex = nextFrames.findIndex(
        ({ nodeId, index }) => nodeId === frame.nodeId && index === frame.index,
      );
      if (selectedIndex >= 0) nextFrames.splice(selectedIndex, 1);
      else nextFrames.push(frame);
    }
    const nextNodeIds = new Set(currentSelection.nodeIds);
    for (const nodeId of touchedNodes) {
      if (nextFrames.some((frame) => frame.nodeId === nodeId))
        nextNodeIds.add(nodeId);
      else nextNodeIds.delete(nodeId);
    }
    return { nodeIds: [...nextNodeIds], frames: nextFrames };
  };
  return {
    onSelect: (nodeId, additive = false) =>
      changeSelection(
        additive ? toggleNodes([nodeId]) : { nodeIds: [nodeId], frames: [] },
      ),
    onSelectFrame: (nodeId, index, additive = false) => {
      const frames = [{ nodeId, index }];
      changeSelection(
        additive ? toggleFrames(frames) : { nodeIds: [nodeId], frames },
      );
    },
    onSelectFrames: (frames, additive = false) => {
      const nodeIds = [...new Set(frames.map(({ nodeId }) => nodeId))];
      changeSelection(additive ? toggleFrames(frames) : { nodeIds, frames });
    },
    onSelectNodes: (nodeIds, additive = false) =>
      changeSelection(
        additive ? toggleNodes(nodeIds) : { nodeIds, frames: [] },
      ),
    onClearSelection: () => changeSelection({ nodeIds: [], frames: [] }),
    onNodePositionChange: (nodeId, position) =>
      onEvent({ type: "node-position.committed", nodeId, position }),
    onReviewResolve: (applied) =>
      onEvent({ type: "generation-review.resolved", applied }),
  };
}
