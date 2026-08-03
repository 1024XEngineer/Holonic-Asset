import { getMockAssetRecord, saveMockAssetRecordRevision } from "./mock";
import type { AssetRecordApi } from "./types";

export const recordApi: AssetRecordApi = {
  get: getMockAssetRecord,
  saveRevision: saveMockAssetRecordRevision,
};

export type {
  AssetRecordApi,
  AssetRecordSaveResult,
  GetAssetRecordInput,
  SaveAssetRecordInput,
} from "./types";
