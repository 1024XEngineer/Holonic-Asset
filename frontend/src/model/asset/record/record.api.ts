import {
  loadCoreAssetWorkspace,
  saveCoreAssetRevision,
} from "./core-asset-record";
import type { AssetWorkspaceApi } from "./types";

export const assetWorkspaceApi: AssetWorkspaceApi = {
  load: loadCoreAssetWorkspace,
  saveRevision: saveCoreAssetRevision,
};

export type {
  AssetWorkspaceApi,
  AssetRecordSaveResult,
  GetAssetRecordInput,
  SaveAssetRecordInput,
} from "./types";
