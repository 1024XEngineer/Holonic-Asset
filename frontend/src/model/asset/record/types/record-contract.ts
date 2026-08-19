import type { AssetRevision } from "../../types";
import type { AssetRecord } from "./asset-record";
import type { AssetWorkspaceData } from "./asset-workspace";

export type GetAssetRecordInput = {
  projectId: string;
  assetId: string;
};

export type SaveAssetRecordInput = GetAssetRecordInput & {
  record: AssetRecord;
  version?: string;
};

export type AssetRecordSaveResult = GetAssetRecordInput & {
  record: AssetRecord;
  version: string;
  history: AssetRevision[];
};

export type AssetWorkspaceApi = {
  load: (input: GetAssetRecordInput) => Promise<AssetWorkspaceData>;
  saveRevision: (input: SaveAssetRecordInput) => Promise<AssetRecordSaveResult>;
};
