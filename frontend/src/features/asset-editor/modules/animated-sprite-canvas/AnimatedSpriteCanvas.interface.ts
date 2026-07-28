import type {
  EditorCharacterAnimation,
  EditorCharacterSpriteSheet,
} from "../../domain";

import type { CanvasPosition } from "./AnimatedSpriteCanvas.constants";
import type { AnimatedSpriteNodeId } from "./animated-sprite-node";

export type AnimatedSpriteCanvasFrameSelection = {
  nodeId: AnimatedSpriteNodeId;
  index: number;
};

export type AnimatedSpriteCanvasSelection = {
  nodeIds: AnimatedSpriteNodeId[];
  frames: AnimatedSpriteCanvasFrameSelection[];
};

export type AnimatedSpriteCanvasModel = {
  prototype: EditorCharacterSpriteSheet;
  animations: EditorCharacterAnimation[];
  unavailableTextureUrls?: ReadonlySet<string>;
  activeDirections?: Readonly<Record<string, AnimatedSpriteNodeId>>;
  nodePositions?: Record<string, CanvasPosition>;
  selection: AnimatedSpriteCanvasSelection;
};

export type AnimatedSpriteCanvasEvent =
  | {
      type: "selection.changed";
      selection: AnimatedSpriteCanvasSelection;
    }
  | {
      type: "node-position.committed";
      nodeId: AnimatedSpriteNodeId;
      position: CanvasPosition;
    }
  | {
      type: "direction.changed";
      nodeId: AnimatedSpriteNodeId;
      directionId: AnimatedSpriteNodeId;
    };

export type AnimatedSpriteCanvasProps = {
  model: AnimatedSpriteCanvasModel;
  onEvent: (event: AnimatedSpriteCanvasEvent) => void;
};
