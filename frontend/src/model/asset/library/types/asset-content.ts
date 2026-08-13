import type { Perspective } from "@/model/project";
import type {
  DirectionCountByPerspective,
  DirectionCountForPerspective,
} from "../../types/perspective-direction";

export type { DirectionCountByPerspective, DirectionCountForPerspective };

export type AssetContentMetadata = Record<string, unknown>;

export type AssetContentBase = {
  metadata?: AssetContentMetadata;
};

export type AssetSizeResponse = {
  width: number;
  height: number;
};

export type AssetPositionResponse = {
  x: number;
  y: number;
};

export type AssetImageResourceResponse = {
  id: number;
  url?: string;
  metadata?: unknown;
};

export type AssetAnimationFrameResponse = AssetImageResourceResponse & {
  duration?: number;
};

export type AssetAnimationResponse = {
  id: number;
  name: string;
  frames: AssetAnimationFrameResponse[];
};

type DirectionalAssetContent<View extends Perspective> = AssetContentBase & {
  directionCount: DirectionCountForPerspective<View>;
  prototype: AssetImageResourceResponse[];
  animations?: AssetAnimationResponse[];
};

export type CharacterAssetContent<View extends Perspective = Perspective> =
  DirectionalAssetContent<View>;

export type ObjectAssetContent<View extends Perspective = Perspective> =
  DirectionalAssetContent<View>;

export type TileSetTileResponse = {
  url?: string;
  position: AssetPositionResponse;
};

export type TileSetItemResponse = {
  name: string;
  tiles?: TileSetTileResponse[];
};

export type TileSetAssetContent = AssetContentBase & {
  items?: TileSetItemResponse[];
};

export type UISetComponentResponse = {
  id: number;
  name: string;
  size: AssetSizeResponse;
  position: AssetPositionResponse;
  anchor?: AssetPositionResponse;
  pivot?: AssetPositionResponse;
  texture?: unknown;
  color?: unknown;
  opacity?: number;
  state?: unknown;
  metadata?: AssetContentMetadata;
};

export type UISetAssetContent = AssetContentBase & {
  components?: UISetComponentResponse[];
};

export type SceneryLayerResponse = {
  id: number;
  name: string;
  resource: string;
  position: AssetPositionResponse;
  transform?: unknown;
  visible?: boolean;
  opacity?: number;
  zIndex?: number;
  metadata?: AssetContentMetadata;
};

export type SceneryAssetContent = AssetContentBase & {
  layers?: SceneryLayerResponse[];
};

export type AudioAssetContent = AssetContentBase;

export type AssetContentByType<View extends Perspective = Perspective> = {
  character: CharacterAssetContent<View>;
  object: ObjectAssetContent<View>;
  tileSet: TileSetAssetContent;
  audio: AudioAssetContent;
  uiset: UISetAssetContent;
  scenery: SceneryAssetContent;
};
