import type { NodeId } from "../animated-sprite-node";
import type { CanvasPosition } from "../AnimatedSpriteCanvas.constants";

export type Bounds = CanvasPosition & { width: number; height: number };

export type AnimatedSpriteSceneSnapshot = {
  readonly positions: Readonly<Record<NodeId, Readonly<CanvasPosition>>>;
  readonly expanded: ReadonlySet<NodeId>;
  readonly playing: ReadonlySet<NodeId>;
  readonly previewFrames: ReadonlyMap<NodeId, number>;
  readonly marquee: {
    readonly start: CanvasPosition;
    readonly end: CanvasPosition;
  } | null;
};

export type AnimatedSpriteSceneState = {
  positions: Record<NodeId, CanvasPosition>;
  expanded: Set<NodeId>;
  playing: Set<NodeId>;
  previewFrames: Map<NodeId, number>;
  marquee: { start: CanvasPosition; end: CanvasPosition } | null;
};
