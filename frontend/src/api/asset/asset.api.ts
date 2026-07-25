import {
  addMockAsset,
  copyMockAsset,
  deleteMockAsset,
  listMockAssetGroups,
  saveMockAssetRevision,
} from "@/api/mock";
import type { ProjectAsset } from "@/types/asset";
import type { AssetKind } from "@/types/asset-kind";
import type { RecordContent } from "@/types/record";

export const assetApi = {
  listGroups: (projectId: string) => listMockAssetGroups(projectId),
  add: (projectId: string, kind: AssetKind, asset: ProjectAsset) =>
    addMockAsset(projectId, kind, asset),
  copy: (projectId: string, assetId: string) =>
    copyMockAsset(projectId, assetId),
  delete: (projectId: string, assetId: string) =>
    deleteMockAsset(projectId, assetId),
  saveRevision: (projectId: string, assetId: string, content: RecordContent) =>
    saveMockAssetRevision(projectId, assetId, content),
};
