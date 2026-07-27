import type { EditorCharacterAnimation } from "@/features/assets/domain";

import type { CanvasPosition } from "./CharacterCanvas.constants";
import type { CharacterCanvasNodeId } from "./character-node";

export type CharacterCanvasFrameSelection = {
  nodeId: CharacterCanvasNodeId;
  index: number;
};

export type CharacterCanvasSelection = {
  nodeIds: CharacterCanvasNodeId[];
  frames: CharacterCanvasFrameSelection[];
};

export type CharacterCanvasModel = {
  animations: EditorCharacterAnimation[];
  nodePositions?: Record<string, CanvasPosition>;
  selection: CharacterCanvasSelection;
};

export type CharacterCanvasEvent =
  | {
      type: "selection.changed";
      selection: CharacterCanvasSelection;
    }
  | {
      type: "node-position.committed";
      nodeId: CharacterCanvasNodeId;
      position: CanvasPosition;
    };

export type CharacterCanvasProps = {
  model: CharacterCanvasModel;
  onEvent: (event: CharacterCanvasEvent) => void;
};
