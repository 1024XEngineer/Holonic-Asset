import type { AssetRevision, AssetRevisionStatus } from "./asset-revision";

export type CharacterSpriteSheet = {
  format: "png-sprite-sheet";
  imageUrl: string;
  frameWidth: number;
  frameHeight: number;
  columns: number;
  rows: number;
  row?: number;
};

export type CharacterAnimationClip = {
  kind: "clip";
  id: string;
  label: string;
  frameCount: number;
  spriteSheet?: CharacterSpriteSheet;
  audio?: { label: string; time: string };
};

export type CharacterAnimation = CharacterAnimationClip;

export type AssetAnimation = {
  id: string;
  name: string;
  frameCount: number;
  status: AssetRevisionStatus;
};

export type SceneryLayer = {
  id: string;
  label: string;
  detail: string;
  imageUrl: string;
  blendMode: "normal" | "multiply";
};

export type SceneryAssetData = { layers: SceneryLayer[] };

export type AssetPreviewFrame = {
  columns: number;
  rows: number;
  column: number;
  row: number;
  frameWidth?: number;
  frameHeight?: number;
  offsetX?: number;
  displayWidth?: string;
};

export type AssetPreviewOffset = { x: string; y: string };

export type AssetPreviewCrop = {
  sourceWidth: number;
  sourceHeight: number;
  x: number;
  y: number;
  width: number;
  height: number;
  displayOffsetY?: string;
};

export type ProjectAsset = {
  id: string;
  name: string;
  description: string;
  version: string;
  canvasSize: string;
  perspective: string;
  tags: string[];
  thumbnailUrl?: string;
  previewCrop?: AssetPreviewCrop;
  previewFrame?: AssetPreviewFrame;
  previewOffset?: AssetPreviewOffset;
  previewScale?: number;
  history: AssetRevision[];
  animations: AssetAnimation[];
  scenery?: SceneryAssetData;
};

export type AssetMetadataUpdate = Pick<
  ProjectAsset,
  "name" | "description" | "tags" | "canvasSize" | "perspective"
>;
