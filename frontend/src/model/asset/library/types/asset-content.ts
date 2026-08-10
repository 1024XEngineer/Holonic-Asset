export type AssetContentMetadata = Record<string, unknown>;

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

export type CharacterAssetContent = {
  directionCount: 2 | 4 | 8;
  prototype: AssetImageResourceResponse[];
  animations?: AssetAnimationResponse[];
};

export type ObjectAssetContent = {
  prototype: AssetImageResourceResponse[];
  animations?: AssetAnimationResponse[];
};

export type TileSetTileResponse = {
  url?: string;
  position: AssetPositionResponse;
};

export type TileSetItemResponse = {
  name: string;
  tiles?: TileSetTileResponse[];
};

export type TileSetAssetContent = {
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

export type UISetAssetContent = {
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

export type SceneryAssetContent = {
  layers?: SceneryLayerResponse[];
};

export type AudioAssetContent = {
  metadata?: AssetContentMetadata;
};

export type AssetContentByType = {
  character: CharacterAssetContent;
  object: ObjectAssetContent;
  tileSet: TileSetAssetContent;
  audio: AudioAssetContent;
  uiset: UISetAssetContent;
  scenery: SceneryAssetContent;
};
