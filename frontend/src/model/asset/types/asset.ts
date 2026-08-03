import type { AssetRevision, AssetRevisionStatus } from "./asset-revision";

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

export type ProjectAsset = {
  id: string;
  name: string;
  description: string;
  version: string;
  canvasSize: string;
  perspective: string;
  tags: string[];
  history: AssetRevision[];
  animations: AssetAnimation[];
  scenery?: SceneryAssetData;
};
