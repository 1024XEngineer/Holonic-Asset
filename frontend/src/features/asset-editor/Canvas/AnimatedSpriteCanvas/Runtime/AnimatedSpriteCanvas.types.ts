import type { Viewport } from "pixi-viewport";
import type { CharacterAnimation } from "@/model";
import type { Rect } from "@/lib/rect";
import type {
  AnimatedSpriteCanvasFrameSelection,
  AnimatedSpriteCanvasModel,
} from "../AnimatedSpriteCanvas.interface";
import type { NodeId } from "../animated-sprite-node";
import type { CanvasPosition } from "../AnimatedSpriteCanvas.constants";

export type AnimatedSpriteCanvasActions = {
  onSelect: (node: NodeId, additive?: boolean) => void;
  onSelectFrame: (node: NodeId, index: number, additive?: boolean) => void;
  onSelectFrames: (
    frames: AnimatedSpriteCanvasFrameSelection[],
    additive?: boolean,
  ) => void;
  onSelectNodes: (nodes: NodeId[], additive?: boolean) => void;
  onClearSelection: () => void;
  onNodePositionChange: (node: NodeId, position: CanvasPosition) => void;
  onReviewResolve: (applied: boolean) => void;
};

export type AnimatedSpriteCanvasRuntimeProps = {
  model: AnimatedSpriteCanvasModel;
  actions: AnimatedSpriteCanvasActions;
  onZoomChange?: (scale: number) => void;
};

export type Bounds = Rect;

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

export type AnimatedSpriteStageContext = {
  viewport: Viewport;
  actions: AnimatedSpriteCanvasActions;
  getAnimations: () => CharacterAnimation[];
  getPrototype: () => { columns: number; rows: number };
  getScene: () => AnimatedSpriteSceneSnapshot;
  getReview: () => AnimatedSpriteCanvasRuntimeProps["model"]["review"];
  moveNode: (node: NodeId, position: CanvasPosition) => void;
  setMarquee: (marquee: AnimatedSpriteSceneState["marquee"]) => void;
  getDragStep: () => number;
  toggleExpanded: (node: NodeId) => void;
  togglePlaying: (node: NodeId) => void;
  render: () => void;
};
