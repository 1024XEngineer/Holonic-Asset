import { getMockAssetRecord, saveMockAssetRecordRevision } from "./mock";
import { loadCoreSpriteAssetWorkspace } from "./core-sprite-record";
import type { AssetRecordApi } from "./types";

export const recordApi: AssetRecordApi = {
  get: async (input) =>
    (await loadCoreSpriteAssetWorkspace(input)) ?? getMockAssetRecord(input),
  saveRevision: saveMockAssetRecordRevision,
};

export type {
  AssetRecordApi,
  AssetRecordSaveResult,
  GetAssetRecordInput,
  SaveAssetRecordInput,
} from "./types";
