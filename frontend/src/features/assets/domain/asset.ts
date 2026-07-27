import type { AssetKind } from "./asset-kind";
import type {
  AssetRecord,
  AssetRecordStatus,
  EditorSceneryLayer,
} from "./record";

export type Asset = {
  id: string;
  name: string;
  kind: AssetKind;
  version: string;
  size: string;
  description: string;
  tags: string[];
  accent: string;
};

export type AssetAnimation = {
  id: string;
  name: string;
  frameCount: number;
  status: AssetRecordStatus;
};

export type SceneryLayer = EditorSceneryLayer;

export type SceneryAssetData = { layers: SceneryLayer[] };

export type ProjectAsset = {
  id: string;
  name: string;
  description: string;
  version: string;
  canvasSize: string;
  perspective: string;
  tags: string[];
  history: AssetRecord[];
  animations: AssetAnimation[];
  scenery?: SceneryAssetData;
};
