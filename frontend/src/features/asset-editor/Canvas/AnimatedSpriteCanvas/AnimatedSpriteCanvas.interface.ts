import type {
  EditorCharacterAnimation,
  EditorCharacterSpriteSheet,
} from "@/model";

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
  nodePositions?: Record<string, CanvasPosition>;
  selection: AnimatedSpriteCanvasSelection;
};
