import type { Viewport } from "pixi-viewport";

import type { EditorCharacterAnimation } from "@/model";
import type { AnimatedSpriteCanvasModel } from "../AnimatedSpriteCanvas.interface";
import type {
  AnimatedSpriteDirectionMap,
  NodeId,
} from "../animated-sprite-node";
import type { CanvasPosition } from "../AnimatedSpriteCanvas.constants";

export type AnimatedSpriteCanvasActions = {
  onSelect: (node: NodeId) => void;
  onSelectFrame: (node: NodeId, index: number) => void;
  onSelectFrames: (node: NodeId, indexes: number[]) => void;
  onSelectNodes: (nodes: NodeId[]) => void;
  onClearSelection: () => void;
  onNodePositionChange: (node: NodeId, position: CanvasPosition) => void;
  onSwitchDirection: (node: NodeId, direction: NodeId) => void;
};

export type AnimatedSpriteCanvasRuntimeProps = {
  model: AnimatedSpriteCanvasModel;
  actions: AnimatedSpriteCanvasActions;
};

export type Bounds = CanvasPosition & { width: number; height: number };

export type AnimatedSpriteSceneSnapshot = {
  readonly positions: Readonly<Record<NodeId, Readonly<CanvasPosition>>>;
  readonly expanded: ReadonlySet<NodeId>;
  readonly playing: ReadonlySet<NodeId>;
  readonly previewFrames: ReadonlyMap<NodeId, number>;
  readonly activeDirections: ReadonlyMap<NodeId, NodeId>;
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
  activeDirections: Map<NodeId, NodeId>;
  marquee: { start: CanvasPosition; end: CanvasPosition } | null;
};

export type AnimatedSpriteStageContext = {
  viewport: Viewport;
  actions: AnimatedSpriteCanvasActions;
  getAnimations: () => EditorCharacterAnimation[];
  getScene: () => AnimatedSpriteSceneSnapshot;
  moveNode: (node: NodeId, position: CanvasPosition) => void;
  setMarquee: (marquee: AnimatedSpriteSceneState["marquee"]) => void;
  getDragStep: () => number;
  toggleExpanded: (node: NodeId) => void;
  togglePlaying: (node: NodeId) => void;
  switchDirection: (node: NodeId) => void;
  getActiveDirections: () => AnimatedSpriteDirectionMap;
  render: () => void;
};
