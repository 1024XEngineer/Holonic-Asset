import { loadCoreAssetWorkspace } from "./core-asset-record";
import { saveCoreSpriteAssetRevision } from "./core-sprite-record";
import { getMockAssetRecord, saveMockAssetRecordRevision } from "./mock";
import type { AssetWorkspaceApi } from "./types";

export const assetWorkspaceApi: AssetWorkspaceApi = {
  load: async (input) =>
    (await loadCoreAssetWorkspace(input)) ?? getMockAssetRecord(input),
  saveRevision: async (input) =>
    (await saveCoreSpriteAssetRevision(input)) ??
    saveMockAssetRecordRevision(input),
};

export type {
  AssetWorkspaceApi,
  AssetRecordSaveResult,
  GetAssetRecordInput,
  SaveAssetRecordInput,
} from "./types";
