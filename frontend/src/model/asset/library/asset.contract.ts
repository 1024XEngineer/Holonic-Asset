import type { components, operations } from "@/model/generated/core-api";
import type {
  AssetContentByType,
  AssetSizeResponse,
} from "./types/asset-content";

export type {
  AssetAnimationFrameResponse,
  AssetAnimationResponse,
  AssetContentBase,
  AssetContentByType,
  AssetContentMetadata,
  AssetImageResourceResponse,
  AssetPositionResponse,
  AssetSizeResponse,
  AudioAssetContent,
  CharacterAssetContent,
  ObjectAssetContent,
  SceneryAssetContent,
  SceneryLayerResponse,
  TileSetAssetContent,
  TileSetItemResponse,
  TileSetTileResponse,
  UISetAssetContent,
  UISetComponentResponse,
} from "./types/asset-content";

type Schemas = components["schemas"];
type GeneratedAssetListItemResponse = Schemas["AssetListItemResponse"];

export type AssetType = GeneratedAssetListItemResponse["type"];

export type TileSetDimensionsResponse = {
  tileSize: AssetSizeResponse;
  tileAmount: { columns: number; rows: number };
};

export type AssetDimensionsByType = {
  character: AssetSizeResponse;
  object: AssetSizeResponse;
  tileSet: TileSetDimensionsResponse;
  audio: null;
  uiset: AssetSizeResponse;
  scenery: AssetSizeResponse;
};

export type AssetDimensions<Type extends AssetType = AssetType> =
  AssetDimensionsByType[Type];

export type AssetContent<Type extends AssetType = AssetType> =
  AssetContentByType[Type & keyof AssetContentByType];

type AssetListItemFields = Omit<
  GeneratedAssetListItemResponse,
  "type" | "dimensions"
>;

export type AssetListItemResponse = {
  [Type in AssetType]: AssetListItemFields & {
    type: Type;
    dimensions: AssetDimensions<Type>;
  };
}[AssetType];

type GeneratedAssetDetailResponse =
  operations["getAsset"]["responses"][200]["content"]["application/json"]["data"];
type AssetDetailFields = Omit<
  GeneratedAssetDetailResponse,
  "type" | "dimensions" | "content"
>;

export type AssetDetailResponse = {
  [Type in AssetType]: AssetDetailFields & {
    type: Type;
    dimensions: AssetDimensions<Type>;
    content?: AssetContent<Type>;
  };
}[AssetType];

export type ListAssetsQuery = NonNullable<
  operations["listAssets"]["parameters"]["query"]
>;

type GeneratedGetAssetsResponse =
  operations["listAssets"]["responses"][200]["content"]["application/json"]["data"];

export type GetAssetsResponse = Omit<GeneratedGetAssetsResponse, "assets"> & {
  assets: AssetListItemResponse[];
};

type GeneratedAssetRecordResponse = Schemas["RecordAssetResponse"];

export type AssetRecordResponse<Type extends AssetType = AssetType> = Omit<
  GeneratedAssetRecordResponse,
  "dimensions" | "content"
> & {
  dimensions: AssetDimensions<Type>;
  content?: AssetContent<Type>;
};

type GeneratedGetAssetRecordsResponse =
  operations["listAssetRecords"]["responses"][200]["content"]["application/json"]["data"];

export type GetAssetRecordsResponse<Type extends AssetType = AssetType> = Omit<
  GeneratedGetAssetRecordsResponse,
  "records"
> & {
  records: AssetRecordResponse<Type>[];
};

export type RecordAssetRequest =
  operations["recordAsset"]["requestBody"]["content"]["application/json"];
export type RecordAssetResponse<Type extends AssetType = AssetType> =
  AssetRecordResponse<Type>;
export type CopyAssetRequest =
  operations["copyAsset"]["requestBody"]["content"]["application/json"];
export type CopyAssetResponse =
  operations["copyAsset"]["responses"][200]["content"]["application/json"]["data"];
export type RollBackAssetRequest =
  operations["rollbackAsset"]["requestBody"]["content"]["application/json"];
export type RollBackAssetResponse =
  operations["rollbackAsset"]["responses"][200]["content"]["application/json"]["data"];

type GeneratedUpdateAssetRequest =
  operations["updateAsset"]["requestBody"]["content"]["application/json"];

export type UpdateAssetRequest<Type extends AssetType = AssetType> = Omit<
  GeneratedUpdateAssetRequest,
  "dimensions"
> & {
  dimensions?: AssetDimensions<Type>;
};

type GeneratedUpdateAssetResponse =
  operations["updateAsset"]["responses"][200]["content"]["application/json"]["data"];
type UpdateAssetFields = Omit<
  GeneratedUpdateAssetResponse,
  "type" | "dimensions"
>;

export type UpdateAssetResponse = {
  [Type in AssetType]: UpdateAssetFields & {
    type: Type;
    dimensions: AssetDimensions<Type>;
  };
}[AssetType];

export type DeleteAssetRequest =
  operations["deleteAsset"]["requestBody"]["content"]["application/json"];
export type DeleteAssetResponse =
  operations["deleteAsset"]["responses"][200]["content"]["application/json"]["data"];
export type AssetMetadataResponse = UpdateAssetResponse;
