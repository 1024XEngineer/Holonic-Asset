import type { AssetRevision } from "../../types";
import type { AssetRecord } from "../../content/types";
import type { AssetWorkspaceData } from "./asset-snapshot";

export type GetAssetRecordInput = {
  projectId: string;
  assetId: string;
};

export type SaveAssetRecordInput = GetAssetRecordInput & {
  record: AssetRecord;
  description?: string;
  version?: string;
};

export type RollbackAssetRecordInput = GetAssetRecordInput & {
  version: string;
};

export type AssetRecordRollbackResult = {
  projectId: string;
  assetId: string;
  version: string;
  contentId: string;
};

export type AssetRecordSaveResult = GetAssetRecordInput & {
  record: AssetRecord;
  version: string;
  history: AssetRevision[];
};

export type AssetRecordApi = {
  load: (input: GetAssetRecordInput) => Promise<AssetWorkspaceData>;
  saveRevision: (input: SaveAssetRecordInput) => Promise<AssetRecordSaveResult>;
  rollback: (
    input: RollbackAssetRecordInput,
  ) => Promise<AssetRecordRollbackResult>;
};

/** @deprecated Use AssetRecordApi. */
export type AssetWorkspaceApi = AssetRecordApi;
