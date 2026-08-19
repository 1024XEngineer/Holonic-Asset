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

export type AnimatedSpriteCanvasReview =
  | {
      kind: "new-animation";
      nodeId: AnimatedSpriteNodeId;
      isResolving: boolean;
    }
  | {
      kind: "comparison";
      nodeId: AnimatedSpriteNodeId;
      candidatePrototype?: CharacterSpriteSheet;
      candidateAnimation?: CharacterAnimation;
      isResolving: boolean;
    };

export type AnimatedSpriteCanvasModel = {
  prototype: CharacterSpriteSheet;
  animations: CharacterAnimation[];
  unavailableTextureUrls?: ReadonlySet<string>;
  nodePositions?: Record<string, CanvasPosition>;
  selection: AnimatedSpriteCanvasSelection;
  review?: AnimatedSpriteCanvasReview;
};

export type AnimatedSpriteCanvasEvent =
  | { type: "selection.changed"; selection: AnimatedSpriteCanvasSelection }
  | {
      type: "node-position.committed";
      nodeId: AnimatedSpriteNodeId;
      position: CanvasPosition;
    }
  | { type: "generation-review.resolved"; applied: boolean };

export type AnimatedSpriteCanvasProps = {
  model: AnimatedSpriteCanvasModel;
  onEvent: (event: AnimatedSpriteCanvasEvent) => void;
};
