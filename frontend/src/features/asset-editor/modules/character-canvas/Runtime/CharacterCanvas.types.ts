import type { Viewport } from "pixi-viewport";

import type { EditorCharacterAnimation } from "@/features/assets/domain";
import type { CharacterCanvasModel } from "../CharacterCanvas.interface";
import type { NodeId } from "../character-node";
import type { CanvasPosition } from "../CharacterCanvas.constants";

export type CharacterCanvasActions = {
  onSelect: (node: NodeId) => void;
  onSelectFrame: (node: NodeId, index: number) => void;
  onSelectFrames: (node: NodeId, indexes: number[]) => void;
  onSelectNodes: (nodes: NodeId[]) => void;
  onClearSelection: () => void;
  onNodePositionChange: (node: NodeId, position: CanvasPosition) => void;
};

export type CharacterCanvasRuntimeProps = {
  model: CharacterCanvasModel;
  actions: CharacterCanvasActions;
};

export type Bounds = CanvasPosition & { width: number; height: number };

export type CharacterSceneSnapshot = {
  readonly positions: Readonly<Record<NodeId, Readonly<CanvasPosition>>>;
  readonly expanded: ReadonlySet<NodeId>;
  readonly playing: ReadonlySet<NodeId>;
  readonly previewFrames: ReadonlyMap<NodeId, number>;
  readonly marquee: {
    readonly start: CanvasPosition;
    readonly end: CanvasPosition;
  } | null;
};

export type CharacterSceneState = {
  positions: Record<NodeId, CanvasPosition>;
  expanded: Set<NodeId>;
  playing: Set<NodeId>;
  previewFrames: Map<NodeId, number>;
  marquee: { start: CanvasPosition; end: CanvasPosition } | null;
};

export type CharacterStageContext = {
  viewport: Viewport;
  actions: CharacterCanvasActions;
  getAnimations: () => EditorCharacterAnimation[];
  getScene: () => CharacterSceneSnapshot;
  moveNode: (node: NodeId, position: CanvasPosition) => void;
  setMarquee: (marquee: CharacterSceneState["marquee"]) => void;
  toggleExpanded: (node: NodeId) => void;
  togglePlaying: (node: NodeId) => void;
  render: () => void;
};
