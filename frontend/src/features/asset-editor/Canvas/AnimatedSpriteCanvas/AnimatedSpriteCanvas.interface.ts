import type { CharacterAnimation, CharacterSpriteSheet } from "@/model";
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
  prototype: CharacterSpriteSheet;
  animations: CharacterAnimation[];
  unavailableTextureUrls?: ReadonlySet<string>;
  nodePositions?: Record<string, CanvasPosition>;
  selection: AnimatedSpriteCanvasSelection;
};

export type AnimatedSpriteCanvasEvent =
  | { type: "selection.changed"; selection: AnimatedSpriteCanvasSelection }
  | {
      type: "node-position.committed";
      nodeId: AnimatedSpriteNodeId;
      position: CanvasPosition;
    };

export type AnimatedSpriteCanvasProps = {
  model: AnimatedSpriteCanvasModel;
  onEvent: (event: AnimatedSpriteCanvasEvent) => void;
};
