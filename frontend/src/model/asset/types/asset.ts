import type { AssetRevision, AssetRevisionStatus } from "./asset-revision";
import { assetTagSchema, type AssetTag } from "./asset-tag";
import { perspectiveSchema, type Perspective } from "@/model/project";
import { z } from "zod";

export type CharacterSpriteSheet = {
  format: "png-sprite-sheet";
  imageUrl: string;
  frameUrls?: string[];
  frameWidth: number;
  frameHeight: number;
  columns: number;
  rows: number;
  row?: number;
};

export type AnimationGenerationConfig = {
  direction: string;
  style?: string;
  action?: string;
  frameCount: number;
  columns: number;
  frameWidth: number;
  frameHeight: number;
  fps: number;
  resolution: string;
  duration: number;
  aspectRatio: string;
};

export type CharacterAnimationClip = {
  kind: "clip";
  id: string;
  groupId?: string;
  label: string;
  frameCount: number;
  frameDurations?: Array<number | undefined>;
  spriteSheet?: CharacterSpriteSheet;
  generation?: AnimationGenerationConfig;
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
  position?: { x: number; y: number };
  transform?: { scale: { x: number; y: number }; rotation: number };
  visible?: boolean;
  opacity?: number;
  zIndex?: number;
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
  perspective: Perspective;
  tags: AssetTag[];
  thumbnailUrl?: string;
  prototypeUrls?: string[];
  previewCrop?: AssetPreviewCrop;
  previewFrame?: AssetPreviewFrame;
  previewOffset?: AssetPreviewOffset;
  previewScale?: number;
  history: AssetRevision[];
  animations: AssetAnimation[];
  scenery?: SceneryAssetData;
};

export const assetMetadataUpdateSchema = z.object({
  name: z.string().trim().min(1, "Asset name is required."),
  description: z.string(),
  tags: z.array(assetTagSchema),
  canvasSize: z.string().trim().min(1, "Canvas size is required."),
  perspective: perspectiveSchema,
});

export type AssetMetadataUpdate = z.infer<typeof assetMetadataUpdateSchema>;
