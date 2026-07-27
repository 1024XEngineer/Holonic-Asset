import type { CharacterCanvasActions } from "./Runtime/CharacterCanvas.types";
import type {
  CharacterCanvasEvent,
  CharacterCanvasSelection,
} from "./CharacterCanvas.interface";

export function createCharacterCanvasActions(
  onEvent: (event: CharacterCanvasEvent) => void,
): CharacterCanvasActions {
  const changeSelection = (selection: CharacterCanvasSelection) => {
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
  };
}
