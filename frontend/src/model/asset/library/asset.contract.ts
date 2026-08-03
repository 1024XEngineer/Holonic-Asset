import type { components, operations } from "@/model/generated/core-api";

type Schemas = components["schemas"];

export type AssetListItemResponse = Schemas["AssetListItemResponse"];
export type AssetType = AssetListItemResponse["type"];
export type AssetDetailResponse =
  operations["getAsset"]["responses"][200]["content"]["application/json"]["data"];
export type ListAssetsQuery = NonNullable<
  operations["listAssets"]["parameters"]["query"]
>;
export type GetAssetsResponse =
  operations["listAssets"]["responses"][200]["content"]["application/json"]["data"];
export type GetAssetRecordsResponse =
  operations["listAssetRecords"]["responses"][200]["content"]["application/json"]["data"];
export type AssetRecordResponse = Schemas["RecordAssetResponse"];
export type RecordAssetRequest =
  operations["recordAsset"]["requestBody"]["content"]["application/json"];
export type RecordAssetResponse =
  operations["recordAsset"]["responses"][200]["content"]["application/json"]["data"];
export type CopyAssetRequest =
  operations["copyAsset"]["requestBody"]["content"]["application/json"];
export type CopyAssetResponse =
  operations["copyAsset"]["responses"][200]["content"]["application/json"]["data"];
export type RollBackAssetRequest =
  operations["rollbackAsset"]["requestBody"]["content"]["application/json"];
export type RollBackAssetResponse =
  operations["rollbackAsset"]["responses"][200]["content"]["application/json"]["data"];
export type UpdateAssetRequest =
  operations["updateAsset"]["requestBody"]["content"]["application/json"];
export type UpdateAssetResponse =
  operations["updateAsset"]["responses"][200]["content"]["application/json"]["data"];
export type DeleteAssetRequest =
  operations["deleteAsset"]["requestBody"]["content"]["application/json"];
export type DeleteAssetResponse =
  operations["deleteAsset"]["responses"][200]["content"]["application/json"]["data"];
export type AssetMetadataResponse = UpdateAssetResponse;
