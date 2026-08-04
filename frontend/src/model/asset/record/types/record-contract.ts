import type { AssetRevision } from "../../types";
import type { AssetRecord } from "./asset-record";
import type { AssetWorkspaceData } from "./asset-workspace";

export type GetAssetRecordInput = {
  projectId: string;
  assetId: string;
};

export type SaveAssetRecordInput = GetAssetRecordInput & {
  record: AssetRecord;
};

export type AssetRecordSaveResult = GetAssetRecordInput & {
  record: AssetRecord;
  version: string;
  history: AssetRevision[];
};

export type AssetRecordApi = {
  get: (input: GetAssetRecordInput) => Promise<AssetWorkspaceData>;
  saveRevision: (input: SaveAssetRecordInput) => Promise<AssetRecordSaveResult>;
};
