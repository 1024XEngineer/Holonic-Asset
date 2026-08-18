import { getMockAssetRecord, saveMockAssetRecordRevision } from "./mock";
import { loadCoreAssetWorkspace } from "./core-asset-record";
import type { AssetRecordApi } from "./types";

export const recordApi: AssetRecordApi = {
  get: async (input) =>
    (await loadCoreAssetWorkspace(input)) ?? getMockAssetRecord(input),
  saveRevision: saveMockAssetRecordRevision,
};

export type {
  AssetRecordApi,
  AssetRecordSaveResult,
  GetAssetRecordInput,
  SaveAssetRecordInput,
} from "./types";
